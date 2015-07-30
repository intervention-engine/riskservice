package assessment

import (
	"github.com/intervention-engine/fhir/models"
	"github.com/intervention-engine/riskservice/fhir"
	"time"
)

func CalculateSimpleRisk(fhirEndpointUrl, patientId string) (*models.RiskAssessment, *Pie, error) {
	patientUrl := fhir.PatientUrl(fhirEndpointUrl, patientId)
	resources := []string{"Conditions", "MedicationStatement"}
	pie := NewPie(patientUrl)
	sum := uint32(0)
	for _, resource := range resources {
		resourceCount, err := fhir.GetCountForPatientResources(fhirEndpointUrl, resource, patientId)
		if err != nil {
			return nil, nil, err
		}
		pie.AddSlice(resource, 50, resourceCount)
		sum += uint32(resourceCount)
	}

	assessment := &models.RiskAssessment{}
	assessment.Subject = &models.Reference{Reference: patientUrl}
	assessment.Date = &models.FHIRDateTime{Time: time.Now(), Precision: models.Timestamp}
	prediction := models.RiskAssessmentPredictionComponent{}
	floatSum := float64(sum)
	prediction.ProbabilityDecimal = &floatSum
	prediction.Outcome = &models.CodeableConcept{Text: "Negative Outcome"}
	assessment.Prediction = []models.RiskAssessmentPredictionComponent{prediction}
	return assessment, pie, nil
}
