package assessment

import (
	"time"

	"github.com/intervention-engine/fhir/models"
	"github.com/intervention-engine/riskservice/fhir"
)

func CalculateSimpleRisk(fhirEndpointUrl, patientId string, ts time.Time) (*models.RiskAssessment, *Pie, error) {
	patientUrl := fhir.PatientUrl(fhirEndpointUrl, patientId)

	pie := NewPie(patientUrl)
	sum := uint32(0)

	conditions, err := fhir.GetPatientConditions(fhir.ResourcesForPatientUrl(fhirEndpointUrl, patientId, "Condition"), ts)
	if err != nil {
		return nil, nil, err
	}
	conditionCount := 0
	for _, condition := range conditions {
		if condition.VerificationStatus == "confirmed" {
			conditionCount++
		}
	}
	pie.AddSlice("Conditions", 50, conditionCount)
	sum += uint32(conditionCount)

	medicationStatements, err := fhir.GetPatientMedicationStatements(fhir.ResourcesForPatientUrl(fhirEndpointUrl, patientId, "MedicationStatement"), ts)
	if err != nil {
		return nil, nil, err
	}
	pie.AddSlice("Medications", 50, len(medicationStatements))
	sum += uint32(len(medicationStatements))

	assessment := &models.RiskAssessment{}
	assessment.Subject = &models.Reference{Reference: patientUrl}
	methodCoding := models.Coding{System: "http://interventionengine.org/risk-assessments", Code: "Simple"}
	assessment.Method = &models.CodeableConcept{Text: "Simple Conditions + Medications", Coding: []models.Coding{methodCoding}}
	assessment.Date = &models.FHIRDateTime{Time: ts, Precision: models.Timestamp}
	prediction := models.RiskAssessmentPredictionComponent{}
	floatSum := float64(sum)
	prediction.ProbabilityDecimal = &floatSum
	prediction.Outcome = &models.CodeableConcept{Text: "Negative Outcome"}
	assessment.Prediction = []models.RiskAssessmentPredictionComponent{prediction}
	return assessment, pie, nil
}
