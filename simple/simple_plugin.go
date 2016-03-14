package simple

import (
	"time"

	"github.com/intervention-engine/fhir/models"
	"github.com/intervention-engine/riskservice/assessment"
	"github.com/intervention-engine/riskservice/plugin"
)

// SimplePlugin is a simple UNPROVEN risk calculation service based on a patient's count of active conditions and
// active medications.  The idea being that the higher this number is, the more likely the patient is to experience
// a negative outcome.  It is a PROOF-OF-CONCEPT only and should NOT be used in any real clinical setting.
type SimplePlugin struct {
}

func NewSimplePlugin() *SimplePlugin {
	return &SimplePlugin{}
}

// Config provides the configuration parameters for the SimplePlugin
func (c *SimplePlugin) Config() plugin.RiskServicePluginConfig {
	return plugin.RiskServicePluginConfig{
		Name: "Simple Conditions + Medications",
		Method: models.CodeableConcept{
			Coding: []models.Coding{{System: "http://interventionengine.org/risk-assessments", Code: "Simple"}},
			Text:   "Simple Conditions + Medications",
		},
		PredictedOutcome: models.CodeableConcept{Text: "Negative Outcome"},
		DefaultPieSlices: []assessment.Slice{
			{Name: "Conditions", Weight: 50, MaxValue: 5},
			{Name: "Medications", Weight: 50, MaxValue: 5},
		},
		RequiredResourceTypes: []string{"Condition", "MedicationStatement"},
	}
}

// Calculate takes a stream of events and returns a slice of corresponding risk calculation results
func (c *SimplePlugin) Calculate(es *plugin.EventStream) ([]plugin.RiskServiceCalculationResult, error) {
	var results []plugin.RiskServiceCalculationResult

	// Keep a map of active conditions and medications -- so we don't double-count duplicates in the record.
	cMap := make(map[string]int)
	mMap := make(map[string]int)

	// Create the initial pie
	pie := assessment.NewPie("Patient/" + es.Patient.Id)
	pie.Slices = c.Config().DefaultPieSlices

	// Now go through the event stream, updating the pie
	for _, event := range es.Events {
		var isFactor bool
		pie = pie.Clone()
		switch r := event.Resource.(type) {
		case models.Condition:
			if r.Code == nil || len(r.Code.Coding) == 0 {
				continue
			}
			isFactor = true
			key := r.Code.Coding[0].System + "|" + r.Code.Coding[0].Code
			count := cMap[key]
			if !event.End {
				cMap[key] = count + 1
			} else if count > 0 {
				cMap[key] = count - 1
			}
			pie.UpdateSliceValue("Conditions", calculateCount(cMap))
		case models.MedicationStatement:
			if r.MedicationCodeableConcept == nil || len(r.MedicationCodeableConcept.Coding) == 0 {
				continue
			}
			isFactor = true
			key := r.MedicationCodeableConcept.Coding[0].System + "|" + r.MedicationCodeableConcept.Coding[0].Code
			count := mMap[key]
			if !event.End {
				mMap[key] = count + 1
			} else if count > 0 {
				mMap[key] = count - 1
			}
			pie.UpdateSliceValue("Medications", calculateCount(mMap))
		}
		if isFactor {
			score := pie.TotalValues()
			results = append(results, plugin.RiskServiceCalculationResult{
				AsOf:               event.Date,
				Score:              &score,
				ProbabilityDecimal: nil,
				Pie:                pie,
			})
		}
	}

	// If there are no results, provide a 0 score for the current time
	if len(results) == 0 {
		zero := 0
		results = append(results, plugin.RiskServiceCalculationResult{
			AsOf:               time.Now(),
			Score:              &zero,
			ProbabilityDecimal: nil,
			Pie:                pie,
		})
	}

	return results, nil
}

// Calculates the count of unique conditions or medications with an upper limit of 5 (maxValue for slice)
func calculateCount(cMap map[string]int) int {
	count := 0
	for _, val := range cMap {
		if val > 0 {
			count++
		}
		if count == 5 {
			break
		}
	}
	return count
}
