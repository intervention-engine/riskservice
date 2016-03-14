package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"gopkg.in/mgo.v2"

	"github.com/intervention-engine/fhir/models"
	"github.com/intervention-engine/riskservice/plugin"
)

type RiskService struct {
	plugins []plugin.RiskServicePlugin
	db      *mgo.Database
}

func NewRiskService(db *mgo.Database) *RiskService {
	return &RiskService{db: db}
}

func (rs *RiskService) RegisterPlugin(plugin plugin.RiskServicePlugin) {
	rs.plugins = append(rs.plugins, plugin)
}

func (rs *RiskService) Calculate(patientId string, fhirEndpoint url.URL, basisPieURL url.URL) error {
	// Build up the query by finding all the resources we must _revinclude
	revIncludeMap := make(map[string]string)
	for _, p := range rs.plugins {
		for _, resource := range p.Config().RequiredResourceTypes {
			switch resource {
			default:
				return fmt.Errorf("Unsupported required resource type: %s", resource)
			case "Condition", "MedicationStatement":
				revIncludeMap[resource] = "patient"
			}
		}
	}
	queryURL, err := url.Parse(fhirEndpoint.String() + "/Patient")
	if err != nil {
		return err
	}
	queryURL.Query().Set("id", patientId)
	for resource, property := range revIncludeMap {
		queryURL.Query().Add("_revinclude", fmt.Sprintf("%s:%s", resource, property))
	}

	// Issue the query and decode into a bundle
	response, err := http.Get(queryURL.String())
	if err != nil {
		return err
	}
	defer response.Body.Close()
	bundle := &models.Bundle{}
	if err = json.NewDecoder(response.Body).Decode(bundle); err != nil {
		return err
	}

	// Convert the bundle into an EventStream
	es, err := plugin.BundleToEventStream(bundle)
	if err != nil {
		return err
	}

	// Now do the calculations for each plugin
	for _, p := range rs.plugins {
		results, err := p.Calculate(es)
		if err != nil {
			return err
		}

		// TODO: sort these results by date

		diff, err := rs.CalculateRiskDiff(patientId, p.Config().Method, results, fhirEndpoint)
		if err != nil {
			return err
		}

		// Build up the bundle with risk assessments to delete and add
		diffBundle := &models.Bundle{}
		diffBundle.Type = "transaction"
		diffBundle.Entry = make([]models.BundleEntryComponent, len(diff.RiskAssessmentsToDelete)+len(diff.RiskCalculationsToAdd))
		for i := range diff.RiskAssessmentsToDelete {
			diffBundle.Entry[i].Request = &models.BundleEntryRequestComponent{
				Method: "DELETE",
				Url:    "RiskAssessment/" + diff.RiskAssessmentsToDelete[i].Id,
			}
		}
		for i := range diff.RiskCalculationsToAdd {
			j := i + len(diff.RiskAssessmentsToDelete)
			diffBundle.Entry[j].Request = &models.BundleEntryRequestComponent{
				Method: "POST",
				Url:    "RiskAssessment",
			}
			diffBundle.Entry[j].Resource = diff.RiskCalculationsToAdd[i].ToRiskAssessment(patientId, basisPieURL, p.Config())
		}

		// Store the new pies
		pieCollection := rs.db.C("pies")
		for i := range diff.RiskCalculationsToAdd {
			err = pieCollection.Insert(diff.RiskCalculationsToAdd[i].Pie)
			if err != nil {
				return err
			}
		}

		// Submit the bundle of assessments to delete and/or add
		data, err := json.Marshal(diffBundle)
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
		for i := range diff.RiskAssessmentsToDelete {
			ra := diff.RiskAssessmentsToDelete[i]
			for _, ref := range ra.Basis {
				if strings.HasPrefix(ref.Reference, basisPieURL.String()+"/") {
					pieID := strings.TrimPrefix(ref.Reference, basisPieURL.String()+"/")
					pieCollection.RemoveId(pieID)
				}
			}
		}
	}

	return nil
}

func (rs *RiskService) CalculateRiskDiff(patientId string, method models.CodeableConcept, results []plugin.RiskServiceCalculationResult, fhirEndpoint url.URL) (*RiskDiff, error) {
	// Build up the query for existing risk assessments
	raURL, err := url.Parse(fhirEndpoint.String() + "/RiskAssessment")
	if err != nil {
		return nil, err
	}
	raURL.Query().Set("patient", patientId)
	if len(method.Coding) == 0 {
		return nil, errors.New("No risk assessment method was specified")
	}
	raURL.Query().Set("method", method.Coding[0].System+"|"+method.Coding[0].Code)
	raURL.Query().Set("_count", "10000") // TODO: Support paging?

	// Get the bundle of existing risk assessments
	response, err := http.Get(raURL.String())
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	raBundle := &models.Bundle{}
	if err = json.NewDecoder(response.Body).Decode(raBundle); err != nil {
		return nil, err
	}

	diff := RiskDiff{}

	// Iterate through the bundle, finding entries that need to be deleted and marking entries that are found
	foundResults := make([]bool, len(results))
	for i := range raBundle.Entry {
		if ra, ok := raBundle.Entry[i].Resource.(*models.RiskAssessment); ok {
			date := ra.Date.Time
			var score *float64
			if len(ra.Prediction) > 0 {
				score = ra.Prediction[0].ProbabilityDecimal
			}

			var found bool
			for j := range results {
				var last bool
				if (j + 1) == len(results) {
					last = true
				}
				result := results[j]
				if result.AsOf.Unix() == date.Unix() && result.GetProbabilityDecimalOrScore() == score {
					if isTaggedMostRecent(ra) != last {
						// Although they are the same, they don't agree on whether they are MOST_RECENT, so don't mark them found!
						break
					}
					found = true
					foundResults[j] = true
					break
				}
			}

			if !found {
				diff.RiskAssessmentsToDelete = append(diff.RiskAssessmentsToDelete)
			}
		}
	}

	// Find the result calculations that weren't found and therefore need to be added
	for i := range foundResults {
		if !foundResults[i] {
			diff.RiskCalculationsToAdd = append(diff.RiskCalculationsToAdd, results[i])
		}
	}

	return &diff, nil
}

type RiskDiff struct {
	RiskCalculationsToAdd   []plugin.RiskServiceCalculationResult
	RiskAssessmentsToDelete []models.RiskAssessment
}

func isTaggedMostRecent(ra *models.RiskAssessment) bool {
	for _, tag := range ra.Meta.Tag {
		if tag.Code == "MOST_RECENT" && tag.System == "http://interventionengine.org/tags/" {
			return true
		}
	}
	return false
}
