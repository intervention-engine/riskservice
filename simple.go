package riskservice

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/intervention-engine/fhir/models"
	"net/http"
	"time"
)

func CalculateSimpleRisk(fhirEndpointUrl, patientId string) (*models.RiskAssessment, error) {
	resources := []string{"Conditions", "MedicationStatement"}
	sum := uint32(0)
	for _, resource := range resources {
		resourceCount, err := GetCountForPatientResources(fhirEndpointUrl, resource, patientId)
		if err != nil {
			return nil, err
		}
		sum += *resourceCount
	}

	assessment := &models.RiskAssessment{}
	assessment.Subject = &models.Reference{Reference: fmt.Sprintf("%s/Patient/%s", fhirEndpointUrl, patientId)}
	assessment.Date = &models.FHIRDateTime{Time: time.Now(), Precision: models.Timestamp}
	prediction := models.RiskAssessmentPredictionComponent{}
	floatSum := float64(sum)
	prediction.ProbabilityDecimal = &floatSum
	prediction.Outcome = &models.CodeableConcept{Text: "Negative Outcome"}
	assessment.Prediction = []models.RiskAssessmentPredictionComponent{prediction}
	return assessment, nil
}

func GetCountForPatientResources(fhirEndpointUrl, resource, patientId string) (*uint32, error) {
	url := fmt.Sprintf("%s/%s?patient:Patient=%s", fhirEndpointUrl, resource, patientId)
	resp, err := http.Get(url)
	defer resp.Body.Close()
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Could not get the %s for patient: %s", resource, patientId))
	}
	decoder := json.NewDecoder(resp.Body)
	bundle := &models.Bundle{}
	err = decoder.Decode(bundle)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Could not decode the %s bundle for patient: %s", resource, patientId))
	}
	return bundle.Total, nil
}
