package service

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"time"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"

	"github.com/intervention-engine/fhir/models"
	"github.com/intervention-engine/riskservice/plugin"
)

// RiskService is an interface for the functions that must be supported by a risk service used in our
// reference implementation risk service server.
type RiskService interface {
	Calculate(patientID string, fhirEndpointURL string, basisPieURL string) error
}

// ReferenceRiskService is a container for risk service plugins that can handle the details of getting data needed
// for the plugins, invoking the calculations on the plugins, posting the new results back to the FHIR server, and
// saving the risk pies to the database.
type ReferenceRiskService struct {
	plugins []plugin.RiskServicePlugin
	db      *mgo.Database
}

// NewReferenceRiskService creates a new risk service backed by the passed in MongoDB instance
func NewReferenceRiskService(db *mgo.Database) *ReferenceRiskService {
	return &ReferenceRiskService{db: db}
}

// RegisterPlugin registers a plugin for use by the risk service
func (rs *ReferenceRiskService) RegisterPlugin(plugin plugin.RiskServicePlugin) {
	rs.plugins = append(rs.plugins, plugin)
}

// Calculate invokes the register plugins to calculate scores for the given patient and post them back to FHIR.
// This deletes all previous risk assessment instances for the patient and replaces them with new instances.
func (rs *ReferenceRiskService) Calculate(patientID string, fhirEndpointURL string, basisPieURL string) error {
	// Get and post the query to retrieve all of the data needed by the risk service plugins
	queryURL, err := rs.getRequiredDataQueryURL(patientID, fhirEndpointURL)
	if err != nil {
		return err
	}
	response, err := http.Get(queryURL)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	bundle := &models.Bundle{}
	if err = json.NewDecoder(response.Body).Decode(bundle); err != nil {
		return err
	}

	// Convert the data bundle and significant birthdays into an EventStream
	es, err := BundleToEventStream(bundle)
	if err != nil {
		return err
	}

	// Now do the calculations for each plugin
	for _, p := range rs.plugins {
		if len(p.Config().Method.Coding) == 0 {
			return errors.New("Risk Assessment Plugins MUST provide a method with a coding")
		}

		// Copy the event stream since we'll add significant birthday events based on plugin config
		esClone := es.Clone()
		addSignificantBirthdayEvents(esClone, p.Config().SignificantBirthdays)

		// Calculate the results
		results, err := p.Calculate(esClone, fhirEndpointURL)
		if err != nil {
			if _, ok := err.(plugin.NotApplicableError); ok {
				continue
			} else {
				return err
			}
		}
		results = sortAndConsolidate(results)

		UpdateRiskAssessmentsAndPies(fhirEndpointURL, patientID, results, rs.db.C("pies"), basisPieURL, p.Config())
	}

	return nil
}

// UpdateRiskAssessmentsAndPies removes existing risk assessments from the FHIR server and replaces them with new ones.
// It also removes old pies from the Mongo database and replaces them with new ones.
func UpdateRiskAssessmentsAndPies(fhirEndpoint string, patientID string, results []plugin.RiskServiceCalculationResult, pieCollection *mgo.Collection, basisPieURL string, config plugin.RiskServicePluginConfig) error {
	// Build up the bundle with risk assessments to delete and add
	raBundle := buildRiskAssessmentBundle(patientID, results, basisPieURL, config)

	// Submit the risk assessment bundle
	data, err := json.Marshal(raBundle)
	if err != nil {
		return err
	}
	response, err := http.Post(fhirEndpoint, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		return fmt.Errorf("Risk assessments did not post properly.  Received response code: %d", response.StatusCode)
	}

	// Delete the old pies
	method := config.Method.Coding[0]
	pieCollection.RemoveAll(bson.M{
		"patient":       fhirEndpoint + "/Patient/" + patientID,
		"method.coding": bson.M{"$elemMatch": bson.M{"system": method.System, "code": method.Code}},
	})

	// Store the new pies along with their method (to identify by patient and method)
	for i := range results {
		method := config.Method
		pieWithMethod := struct {
			plugin.Pie `bson:",inline"`
			Method     *models.CodeableConcept `bson:"method"`
		}{
			*results[i].Pie,
			&method,
		}
		if err = pieCollection.Insert(&pieWithMethod); err != nil {
			return err
		}
	}
	return nil
}

// getRequiredDataQueryURL constructs the URL to use for identifying all risk assessments for a given patient
// using a given method.  This is used to delete the old set of assessments before adding the new set.
func (rs *ReferenceRiskService) getRequiredDataQueryURL(patientID, fhirEndpointURL string) (string, error) {
	// Build up the query by finding all the resources we must _revinclude
	revIncludeMap := make(map[string]string)
	for _, p := range rs.plugins {
		for _, resource := range p.Config().RequiredResourceTypes {
			switch resource {
			default:
				return "", fmt.Errorf("Unsupported required resource type: %s", resource)
			// NOTE: This only supports those resources we currently need in our reference implementation plugins
			case "Condition", "MedicationStatement":
				revIncludeMap[resource] = "patient"
			}
		}
	}
	queryURL, err := url.Parse(fhirEndpointURL + "/Patient")
	if err != nil {
		return "", err
	}
	params := url.Values{}
	params.Set("_id", patientID)
	for resource, property := range revIncludeMap {
		params.Add("_revinclude", fmt.Sprintf("%s:%s", resource, property))
	}
	queryURL.RawQuery = params.Encode()

	return queryURL.String(), nil
}

// BundleToEventStream takes a bundle of resources and converts them to an EventStream.  Currently only a
// limited set of resource types are supported, with unsupported resource types resulting in an error.  If
// the bundle contains more than one patient, this is also considered an error.
func BundleToEventStream(bundle *models.Bundle) (es *plugin.EventStream, err error) {
	var patient *models.Patient
	events := make([]plugin.Event, 0, len(bundle.Entry))
	for _, entry := range bundle.Entry {
		switch r := entry.Resource.(type) {
		default:
			err = fmt.Errorf("Unsupported: Converting %s to Event", reflect.TypeOf(r).Elem().Name())
			return
		case *models.Patient:
			if patient != nil {
				err = errors.New("Found more than one patient in resources")
				return
			}
			patient = r
		case *models.Condition:
			if r.VerificationStatus != "confirmed" {
				continue
			}
			if onset, err := findDate(false, r.OnsetDateTime, r.OnsetPeriod, r.DateRecorded); err == nil {
				events = append(events, plugin.Event{Date: onset, Type: "Condition", End: false, Value: r})
			}
			if abatement, err := findDate(true, r.AbatementDateTime, r.AbatementPeriod); err == nil {
				events = append(events, plugin.Event{Date: abatement, Type: "Condition", End: true, Value: r})
			}
			// TODO: What happens if there is no date at all?
		case *models.MedicationStatement:
			if r.Status == "" || r.Status == "entered-in-error" {
				continue
			}
			if active, err := findDate(false, r.EffectiveDateTime, r.EffectivePeriod, r.DateAsserted); err == nil {
				events = append(events, plugin.Event{Date: active, Type: "MedicationStatement", End: false, Value: r})
			}
			if inactive, err := findDate(true, r.EffectivePeriod); err == nil {
				events = append(events, plugin.Event{Date: inactive, Type: "MedicationStatement", End: true, Value: r})
			}
			// TODO: What happens if there is no date at all?
		case *models.Observation:
			if r.Status != "final" && r.Status != "amended" && r.Status != "preliminary" && r.Status != "registered" {
				continue
			}
			if effective, err := findDate(false, r.EffectiveDateTime, r.EffectivePeriod, r.Issued); err == nil {
				events = append(events, plugin.Event{Date: effective, Type: "Observation", End: false, Value: r})
			}
			if ineffective, err := findDate(true, r.EffectivePeriod); err == nil {
				events = append(events, plugin.Event{Date: ineffective, Type: "Observation", End: true, Value: r})
			}
			// TODO: What happens if there is no date at all?
		}
	}
	es = plugin.NewEventStream(patient)
	plugin.SortEventsByDate(events)
	es.Events = events
	return es, nil
}

func addSignificantBirthdayEvents(es *plugin.EventStream, birthdays []int) {
	if len(birthdays) == 0 || es.Patient == nil || es.Patient.BirthDate == nil {
		return
	}

	for _, age := range birthdays {
		bd := es.Patient.BirthDate.Time.AddDate(age, 0, 0)
		if bd.Before(time.Now()) {
			es.Events = append(es.Events, plugin.Event{Date: bd, Type: "Age", End: false, Value: age})
		}
	}

	plugin.SortEventsByDate(es.Events)
}

func findDate(usePeriodEnd bool, datesAndPeriods ...interface{}) (time.Time, error) {
	for _, t := range datesAndPeriods {
		switch t := t.(type) {
		case models.FHIRDateTime:
			return t.Time, nil
		case *models.FHIRDateTime:
			if t != nil {
				return t.Time, nil
			}
		case models.Period:
			if !usePeriodEnd && t.Start != nil {
				return t.Start.Time, nil
			} else if usePeriodEnd && t.End != nil {
				return t.End.Time, nil
			}
		case *models.Period:
			if !usePeriodEnd && t != nil && t.Start != nil {
				return t.Start.Time, nil
			} else if usePeriodEnd && t != nil && t.End != nil {
				return t.End.Time, nil
			}
		}
	}

	return time.Time{}, errors.New("No date found")
}

func buildRiskAssessmentBundle(patientID string, results []plugin.RiskServiceCalculationResult, basisPieURL string, config plugin.RiskServicePluginConfig) *models.Bundle {
	raBundle := &models.Bundle{}
	raBundle.Type = "transaction"
	raBundle.Entry = make([]models.BundleEntryComponent, len(results)+1)
	raBundle.Entry[0].Request = &models.BundleEntryRequestComponent{
		Method: "DELETE",
		Url:    getRiskAssessmentDeleteURL(config.Method, patientID),
	}
	for i := range results {
		raBundle.Entry[i+1].Request = &models.BundleEntryRequestComponent{
			Method: "POST",
			Url:    "RiskAssessment",
		}
		ra := results[i].ToRiskAssessment(patientID, basisPieURL, config)
		if (i + 1) == len(results) {
			ra.Meta = &models.Meta{
				Tag: []models.Coding{{System: "http://interventionengine.org/tags/", Code: "MOST_RECENT"}},
			}
		}
		raBundle.Entry[i+1].Resource = ra
	}
	return raBundle
}

// getRiskAssessmentDeleteURL constructs the URL to use for identifying all risk assessments for a given patient
// using a given method.  This is used to delete the old set of assessments before adding the new set.
func getRiskAssessmentDeleteURL(concept models.CodeableConcept, patientID string) string {
	params := url.Values{}
	params.Set("method", fmt.Sprintf("%s|%s", concept.Coding[0].System, concept.Coding[0].Code))
	params.Set("patient", patientID)
	return fmt.Sprintf("RiskAssessment?%s", params.Encode())
}

// sortAndConsolidate sorts calculations by date and then consolidates the ones that have the same timestamp into one,
// choosing whichever was last in the original order
func sortAndConsolidate(results []plugin.RiskServiceCalculationResult) []plugin.RiskServiceCalculationResult {
	// Use stable sort to retain original order on equal elements
	plugin.SortResultsByAsOfDate(results)
	for i := 0; i < len(results); i++ {
		if i > 0 && results[i].AsOf.Equal(results[i-1].AsOf) {
			results = append(results[:(i-1)], results[i:]...)
			i--
		}
	}
	return results
}
