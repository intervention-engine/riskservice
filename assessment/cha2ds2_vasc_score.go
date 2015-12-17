package assessment

import (
	"strings"
	"time"

	"github.com/intervention-engine/fhir/models"
	"github.com/intervention-engine/riskservice/fhir"
)

type CHADCondition struct {
	Name   string
	Code   string
	System string
	Points int
}

const PieSliceWidth = 11 // 9 pie slices out of a total value of 100

// Taken from the Wikipedia page
// Maps the CHADS score to the annual stroke risk
var ScoreToStrokeRisk = map[int]float64{0: 0, 1: 1.3, 2: 2.2, 3: 3.2, 4: 4.0, 5: 6.7, 6: 9.8, 7: 9.6, 8: 12.5, 9: 15.2}

// Assumes ICD-9 and 1 point
func NewCHAD(name, code string) CHADCondition {
	return CHADCondition{Name: name, Code: code, System: "http://hl7.org/fhir/sid/icd-9", Points: 1}
}

// An implementation of https://en.wikipedia.org/wiki/CHA2DS2%E2%80%93VASc_score
func CalculateCHADSRisk(fhirEndpointUrl, patientId string, ts time.Time) (*models.RiskAssessment, *Pie, error) {
	patientUrl := fhir.PatientUrl(fhirEndpointUrl, patientId)
	pie := NewPie(patientUrl)
	conditions, conditionErr := fhir.GetPatientConditions(fhir.ResourcesForPatientUrl(fhirEndpointUrl, patientId, "Condition"), ts)
	if conditionErr != nil {
		return nil, nil, conditionErr
	}
	patient, patientErr := fhir.GetPatient(fhir.PatientUrl(fhirEndpointUrl, patientId))
	if patientErr != nil {
		return nil, nil, patientErr
	}
	chadScore := CalculateConditionPortion(conditions, pie)
	chadScore += CalculateDemographicPortion(patient, pie, ts)

	assessment := &models.RiskAssessment{}
	assessment.Subject = &models.Reference{Reference: patientUrl}
	methodCoding := models.Coding{System: "http://interventionengine.org/risk-assessments", Code: "CHADS"}
	assessment.Method = &models.CodeableConcept{Text: "CHA2DS2–VASc score", Coding: []models.Coding{methodCoding}}
	assessment.Date = &models.FHIRDateTime{Time: ts, Precision: models.Timestamp}
	prediction := models.RiskAssessmentPredictionComponent{}
	strokeRisk := ScoreToStrokeRisk[chadScore]
	prediction.ProbabilityDecimal = &strokeRisk
	prediction.Outcome = &models.CodeableConcept{Text: "Stroke"}
	assessment.Prediction = []models.RiskAssessmentPredictionComponent{prediction}
	return assessment, pie, nil
}

func CalculateDemographicPortion(patient *models.Patient, pie *Pie, ts time.Time) int {
	chadScore := 0
	if patient.Gender == "female" {
		pie.AddSlice("Gender", PieSliceWidth, 1, 1)
		chadScore += 1
	} else {
		pie.AddSlice("Gender", PieSliceWidth, 0, 1)
	}
	age := Age(patient, ts)
	switch {
	case age >= 65 && age < 75:
		pie.AddSlice("Age", PieSliceWidth*2, 1, 2)
		chadScore++
	case age >= 75:
		pie.AddSlice("Age", PieSliceWidth*2, 2, 2)
		chadScore += 2
	default:
		pie.AddSlice("Age", PieSliceWidth*2, 0, 2)
	}
	return chadScore
}

func CalculateConditionPortion(patientConditions []*models.Condition, pie *Pie) int {
	conditions := []CHADCondition{NewCHAD("Congestive Heart Failure", "428")}
	conditions = append(conditions, NewCHAD("Hypertension", "401"))
	conditions = append(conditions, NewCHAD("Diabetes", "250"))
	stroke := NewCHAD("Stroke", "434")
	stroke.Points = 2
	conditions = append(conditions, stroke)
	conditions = append(conditions, NewCHAD("Vascular Disease", "443"))

	chadScore := 0

	for _, condition := range conditions {
		weight := condition.Points * PieSliceWidth
		value := 0
		if FuzzyFindInConditions(condition.Code, condition.System, patientConditions) {
			value = condition.Points
		}
		chadScore += value
		pie.AddSlice(condition.Name, weight, value, condition.Points)
	}

	return chadScore
}

func FuzzyFindInConditions(codeStart, codeSystem string, conditions []*models.Condition) bool {
	for _, condition := range conditions {
		if condition.VerificationStatus == "confirmed" {
			for _, coding := range condition.Code.Coding {
				if strings.HasPrefix(coding.Code, codeStart) && coding.System == codeSystem {
					return true
				}
			}
		}
	}
	return false
}

func Age(patient *models.Patient, ts time.Time) int {
	patientBirthDay := patient.BirthDate.Time
	age := ts.Year() - patientBirthDay.Year()
	if patientBirthDay.YearDay() > ts.YearDay() {
		age--
	}
	return age
}
