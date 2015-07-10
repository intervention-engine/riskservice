package riskservice

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"github.com/intervention-engine/fhir/models"
	"net/http"
	"time"
)

func CalculateSimpleRisk(fhirEndpointUrl, patientId string) (*models.RiskAssessment, *Pie, error) {
	patientUrl := fmt.Sprintf("%s/Patient/%s", fhirEndpointUrl, patientId)
	resources := []string{"Conditions", "MedicationStatement"}
	pie := NewPie(patientUrl)
	sum := uint32(0)
	for _, resource := range resources {
		resourceCount, err := GetCountForPatientResources(fhirEndpointUrl, resource, patientId)
		if err != nil {
			return nil, nil, err
		}
		pie.AddSlice(resource, 50, int(*resourceCount))
		sum += *resourceCount
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
		spew.Dump(err)
		return nil, errors.New(fmt.Sprintf("Could not decode the %s bundle for patient: %s", resource, patientId))
	}
	return bundle.Total, nil
}
