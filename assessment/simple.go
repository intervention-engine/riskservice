package assessment

import (
	"github.com/intervention-engine/fhir/models"
	"github.com/intervention-engine/riskservice/fhir"
	"time"
)

func CalculateSimpleRisk(fhirEndpointUrl, patientId string, ts time.Time) (*models.RiskAssessment, *Pie, error) {
	patientUrl := fhir.PatientUrl(fhirEndpointUrl, patientId)

	pie := NewPie(patientUrl)
	sum := uint32(0)

	conditions, err := fhir.GetPatientConditions(fhir.ResourcesForPatientUrl(fhirEndpointUrl, patientId, "Condition"), ts)
	if err != nil {
		return nil, nil, err
	}
	pie.AddSlice("Conditions", 50, len(conditions))
	sum += uint32(len(conditions))

	medicationStatements, err := fhir.GetPatientMedicationStatements(fhir.ResourcesForPatientUrl(fhirEndpointUrl, patientId, "MedicationStatement"), ts)
	if err != nil {
		return nil, nil, err
	}
	pie.AddSlice("Medications", 50, len(medicationStatements))
	sum += uint32(len(medicationStatements))

	assessment := &models.RiskAssessment{}
	assessment.Subject = &models.Reference{Reference: patientUrl}
	assessment.Date = &models.FHIRDateTime{Time: ts, Precision: models.Timestamp}
	prediction := models.RiskAssessmentPredictionComponent{}
	floatSum := float64(sum)
	prediction.ProbabilityDecimal = &floatSum
	prediction.Outcome = &models.CodeableConcept{Text: "Negative Outcome"}
	assessment.Prediction = []models.RiskAssessmentPredictionComponent{prediction}
	return assessment, pie, nil
}
