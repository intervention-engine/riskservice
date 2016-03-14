package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sort"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"

	"github.com/intervention-engine/fhir/models"
	"github.com/intervention-engine/riskservice/assessment"
	"github.com/intervention-engine/riskservice/plugin"
)

// RiskService is a container for risk service plugins that can handle the details of getting data needed for the
// plugins, invoking the calculations on the plugins, posting the new results back to the FHIR server, and saving
// the risk pies to the database.
type RiskService struct {
	plugins []plugin.RiskServicePlugin
	db      *mgo.Database
}

// NewRiskService creates a new risk service backed by the passed in MongoDB instance
func NewRiskService(db *mgo.Database) *RiskService {
	return &RiskService{db: db}
}

// RegisterPlugin registers a plugin for use by the risk service
func (rs *RiskService) RegisterPlugin(plugin plugin.RiskServicePlugin) {
	rs.plugins = append(rs.plugins, plugin)
}

// Calculate invokes the register plugins to calculate scores for the given patient and post them back to FHIR.
// This deletes all previous risk assessment instances for the patient and replaces them with new instances.
func (rs *RiskService) Calculate(patientID string, fhirEndpoint url.URL, basisPieURL url.URL) error {
	// Get and post the query to retrieve all of the data needed by the risk service plugins
	queryURL, err := rs.getRequiredDataQueryURL(patientID, fhirEndpoint)
	if err != nil {
		return err
	}
	response, err := http.Get(queryURL.String())
	if err != nil {
		return err
	}
	defer response.Body.Close()
	bundle := &models.Bundle{}
	if err = json.NewDecoder(response.Body).Decode(bundle); err != nil {
		return err
	}

	// Convert the data bundle into an EventStream
	es, err := plugin.BundleToEventStream(bundle)
	if err != nil {
		return err
	}

	// Now do the calculations for each plugin
	for _, p := range rs.plugins {

		if len(p.Config().Method.Coding) == 0 {
			return errors.New("Risk Assessment Plugins MUST provide a method with a coding")
		}

		results, err := p.Calculate(es)
		if err != nil {
			if _, ok := err.(plugin.NotApplicableError); ok {
				continue
			} else {
				return err
			}
		}
		results = sortAndConsolidate(results)

		// Build up the bundle with risk assessments to delete and add
		raBundle := buildRiskAssessmentBundle(patientID, results, basisPieURL, p)

		// Store the new pies along with their method (to identify by patient and method)
		pieCollection := rs.db.C("pies")
		for i := range results {
			method := p.Config().Method
			pieWithMethod := struct {
				assessment.Pie `bson:",inline"`
				Method         *models.CodeableConcept `bson:"method"`
			}{
				*results[i].Pie,
				&method,
			}
			if err = pieCollection.Insert(&pieWithMethod); err != nil {
				return err
			}
		}

		// Submit the risk assessment bundle
		data, err := json.Marshal(raBundle)
		if err != nil {
			return err
		}
		response, err = http.Post(fhirEndpoint.String(), "application/json", bytes.NewBuffer(data))
		if err != nil {
			return err
		}
		defer response.Body.Close()

		if response.StatusCode != 200 {
			return fmt.Errorf("Risk assessments did not post properly.  Received response code: %d", response.StatusCode)
		}

		// Delete the old pies
		method := p.Config().Method.Coding[0]
		pieCollection.RemoveAll(bson.M{
			"patient":       patientID,
			"method.coding": bson.M{"$elemMatch": bson.M{"system": method.System, "code": method.Code}},
		})
	}

	return nil
}

// getRiskAssessmentDeleteURL constructs the URL to use for identifying all risk assessments for a given patient
// using a given method.  This is used to delete the old set of assessments before adding the new set.
func (rs *RiskService) getRequiredDataQueryURL(patientID string, fhirEndpoint url.URL) (queryURL *url.URL, err error) {
	// Build up the query by finding all the resources we must _revinclude
	revIncludeMap := make(map[string]string)
	for _, p := range rs.plugins {
		for _, resource := range p.Config().RequiredResourceTypes {
			switch resource {
			default:
				err = fmt.Errorf("Unsupported required resource type: %s", resource)
				return
			// NOTE: This only supports those resources we currently need in our reference implementation plugins
			case "Condition", "MedicationStatement":
				revIncludeMap[resource] = "patient"
			}
		}
	}
	if queryURL, err = url.Parse(fhirEndpoint.String() + "/Patient"); err == nil {
		params := url.Values{}
		params.Set("id", patientID)
		for resource, property := range revIncludeMap {
			params.Add("_revinclude", fmt.Sprintf("%s:%s", resource, property))
		}
		queryURL.RawQuery = params.Encode()
	}
	return
}

func buildRiskAssessmentBundle(patientID string, results []plugin.RiskServiceCalculationResult, basisPieURL url.URL, p plugin.RiskServicePlugin) *models.Bundle {
	raBundle := &models.Bundle{}
	raBundle.Type = "transaction"
	raBundle.Entry = make([]models.BundleEntryComponent, len(results)+1)
	raBundle.Entry[0].Request = &models.BundleEntryRequestComponent{
		Method: "DELETE",
		Url:    getRiskAssessmentDeleteURL(p.Config().Method, patientID),
	}
	for i := range results {
		raBundle.Entry[i+1].Request = &models.BundleEntryRequestComponent{
			Method: "POST",
			Url:    "RiskAssessment",
		}
		ra := results[i].ToRiskAssessment(patientID, basisPieURL, p.Config())
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
	sort.Stable(byAsOfDate(results))
	for i := 0; i < len(results); i++ {
		if i > 0 && results[i].AsOf.Equal(results[i-1].AsOf) {
			results = append(results[:(i-1)], results[i:]...)
			i--
		}
	}
	return results
}

type byAsOfDate []plugin.RiskServiceCalculationResult

func (d byAsOfDate) Len() int {
	return len(d)
}
func (d byAsOfDate) Swap(i, j int) {
	d[i], d[j] = d[j], d[i]
}
func (d byAsOfDate) Less(i, j int) bool {
	return d[i].AsOf.Before(d[j].AsOf)
}
