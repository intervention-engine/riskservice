package assessments

import (
	"strings"
	"time"

	"github.com/intervention-engine/fhir/models"
	"github.com/intervention-engine/riskservice/plugin"
)

// CHA2DS2VAScPlugin is a risk calculation service implementing the CHA2DS2-VASc Score for Stroke in Patients with
// Atrial Fibrillation: https://en.wikipedia.org/wiki/CHA2DS2%E2%80%93VASc_score
type CHA2DS2VAScPlugin struct {
}

// NewCHA2DS2VAScPlugin returns a new CHA2DS2VAScPlugin
func NewCHA2DS2VAScPlugin() *CHA2DS2VAScPlugin {
	return &CHA2DS2VAScPlugin{}
}

// Config provides the configuration parameters for the CHA2DS2VAScPlugin
func (c *CHA2DS2VAScPlugin) Config() plugin.RiskServicePluginConfig {
	return plugin.RiskServicePluginConfig{
		Name: "CHA2DS2–VASc score",
		Method: models.CodeableConcept{
			Coding: []models.Coding{{System: "http://interventionengine.org/risk-assessments", Code: "CHADS"}},
			Text:   "CHA2DS2–VASc score",
		},
		PredictedOutcome: models.CodeableConcept{Text: "Stroke"},
		DefaultPieSlices: []plugin.Slice{
			{Name: "Congestive Heart Failure", Weight: 11, MaxValue: 1},
			{Name: "Hypertension", Weight: 11, MaxValue: 1},
			{Name: "Diabetes", Weight: 11, MaxValue: 1},
			{Name: "Stroke", Weight: 22, MaxValue: 2},
			{Name: "Vascular Disease", Weight: 11, MaxValue: 1},
			{Name: "Age", Weight: 22, MaxValue: 2},
			{Name: "Gender", Weight: 11, MaxValue: 1},
		},
		RequiredResourceTypes: []string{"Condition"},
		SignificantBirthdays:  []int{65, 75},
	}
}

// Calculate takes a stream of events and returns a slice of corresponding risk calculation results
func (c *CHA2DS2VAScPlugin) Calculate(es *plugin.EventStream, fhirEndpointURL string) ([]plugin.RiskServiceCalculationResult, error) {
	var results []plugin.RiskServiceCalculationResult

	// COMMENTED OUT because frontend does not properly handle patients w/ no RiskAssessments,
	// so we must proceed with at least on RiskAssessment of score 0 until "not applicable" is supported
	//
	// // First make sure there is AFIB in the history, since this score is only valid for patients with AFIB
	// var hasAFib bool
	// for i := 0; !hasAFib && i < len(es.Events); i++ {
	// 	if es.Events[i].Type == "Condition" && !es.Events[i].End {
	// 		if cond, ok := es.Events[i].Value.(*models.Condition); ok {
	// 			hasAFib = fuzzyFindCondition("427.31", "http://hl7.org/fhir/sid/icd-9", cond)
	// 		}
	// 	}
	// }
	// if !hasAFib {
	// 	return nil, plugin.NewNotApplicableError("CHA2DS2-VASc is only applicable to patients with Atrial Fibrillation")
	// }

	// Create the initial pie based on gender
	pie := plugin.NewPie(fhirEndpointURL + "/Patient/" + es.Patient.Id)
	pie.Slices = c.Config().DefaultPieSlices
	if es.Patient.Gender == "female" {
		pie.UpdateSliceValue("Gender", 1)
	} else if es.Patient.Gender == "male" {
		pie.UpdateSliceValue("Gender", 0)
	}

	// Now go through the event stream, updating the pie
	var hasAfib bool
	for _, event := range es.Events {
		// NOTE: We are not paying attention to end times -- if it's in the patient history, we count it.
		// Also guard against future dates (for example, our patient generator can create future events)
		if event.End || event.Date.Local().After(time.Now()) {
			continue
		}

		var isFactor bool
		pie = pie.Clone(true)
		switch event.Type {
		case "Condition":
			r := event.Value.(*models.Condition)
			if fuzzyFindCondition("427.31", "http://hl7.org/fhir/sid/icd-9", r) {
				// Found atrial fibrillation, so all events from here on should produce scores
				hasAfib = true
				isFactor = true
			} else if fuzzyFindCondition("428", "http://hl7.org/fhir/sid/icd-9", r) {
				pie.UpdateSliceValue("Congestive Heart Failure", 1)
				isFactor = true
			} else if fuzzyFindCondition("401", "http://hl7.org/fhir/sid/icd-9", r) {
				pie.UpdateSliceValue("Hypertension", 1)
				isFactor = true
			} else if fuzzyFindCondition("250", "http://hl7.org/fhir/sid/icd-9", r) {
				pie.UpdateSliceValue("Diabetes", 1)
				isFactor = true
			} else if fuzzyFindCondition("434", "http://hl7.org/fhir/sid/icd-9", r) {
				pie.UpdateSliceValue("Stroke", 2)
				isFactor = true
			} else if fuzzyFindCondition("443", "http://hl7.org/fhir/sid/icd-9", r) {
				pie.UpdateSliceValue("Vascular Disease", 1)
				isFactor = true
			}
		case "Age":
			age := event.Value.(int)
			if age >= 65 && age < 75 {
				pie.UpdateSliceValue("Age", 1)
				isFactor = true
			} else if age >= 75 {
				pie.UpdateSliceValue("Age", 2)
				isFactor = true
			}
		}
		if hasAfib && isFactor {
			score := pie.TotalValues()
			percent := ScoreToStrokeRisk[score]
			results = append(results, plugin.RiskServiceCalculationResult{
				AsOf:               event.Date,
				Score:              &score,
				ProbabilityDecimal: &percent,
				Pie:                pie,
			})
		}
	}

	// If there are no results, provide a 0 score for the current time
	if len(results) == 0 {
		zero := 0
		zeroFlt := 0.0
		defaultPie := plugin.NewPie(fhirEndpointURL + "/Patient/" + es.Patient.Id)
		defaultPie.Slices = c.Config().DefaultPieSlices
		results = append(results, plugin.RiskServiceCalculationResult{
			AsOf:               time.Now(),
			Score:              &zero,
			ProbabilityDecimal: &zeroFlt,
			Pie:                defaultPie,
		})
	}

	return results, nil
}

// ScoreToStrokeRisk maps the CHA2DS2-VASc score to the annual stroke risk
// See: http://stroke.ahajournals.org/content/41/12/2731/T4.expansion.html
var ScoreToStrokeRisk = map[int]float64{0: 0, 1: 1.3, 2: 2.2, 3: 3.2, 4: 4.0, 5: 6.7, 6: 9.8, 7: 9.6, 8: 6.7, 9: 15.2}

func fuzzyFindCondition(codeStart, codeSystem string, condition *models.Condition) bool {
	if condition.VerificationStatus == "confirmed" {
		for _, coding := range condition.Code.Coding {
			if strings.HasPrefix(coding.Code, codeStart) && coding.System == codeSystem {
				return true
			}
		}
	}
	return false
}
