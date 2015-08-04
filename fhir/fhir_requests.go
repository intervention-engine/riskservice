package fhir

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/intervention-engine/fhir/models"
	"net/http"
	"time"
)

func GetCountForPatientResources(fhirEndpointUrl, resource, patientId string) (int, error) {
	url := ResourcesForPatientUrl(fhirEndpointUrl, patientId, resource)
	return GetCount(url)
}

func GetCount(fullFhirUrl string) (int, error) {
	resp, err := http.Get(fullFhirUrl)
	defer resp.Body.Close()
	if err != nil {
		return 0, errors.New(fmt.Sprintf("Could not get: %s", fullFhirUrl))
	}
	decoder := json.NewDecoder(resp.Body)
	bundle := &models.Bundle{}
	err = decoder.Decode(bundle)
	if err != nil {
		return 0, errors.New(fmt.Sprintf("Could not decode: %s", fullFhirUrl))
	}
	total := bundle.Total
	return int(*total), nil
}

func GetPatientConditions(fullFhirUrl string, ts time.Time) ([]models.Condition, error) {
	resp, err := http.Get(fullFhirUrl)
	defer resp.Body.Close()
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Could not get: %s", fullFhirUrl))
	}
	decoder := json.NewDecoder(resp.Body)
	bundle := &models.Bundle{}
	err = decoder.Decode(bundle)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Could not decode: %s", fullFhirUrl))
	}
	var conditions []models.Condition
	for _, resource := range bundle.Entry {
		c, ok := resource.Resource.(models.Condition)
		if ok {
			if c.OnsetDateTime == nil || (c.OnsetDateTime != nil && c.OnsetDateTime.Time.After(ts)) {
				conditions = append(conditions, c)
			}
		}
	}
	return conditions, nil
}

func GetPatientMedicationStatements(fullFhirUrl string, ts time.Time) ([]models.MedicationStatement, error) {
	resp, err := http.Get(fullFhirUrl)
	defer resp.Body.Close()
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Could not get: %s", fullFhirUrl))
	}
	decoder := json.NewDecoder(resp.Body)
	bundle := &models.Bundle{}
	err = decoder.Decode(bundle)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Could not decode: %s", fullFhirUrl))
	}
	var medicationStatements []models.MedicationStatement
	for _, resource := range bundle.Entry {
		ms, ok := resource.Resource.(models.MedicationStatement)
		if ok {
			if ms.EffectiveDateTime == nil || (ms.EffectiveDateTime != nil && ms.EffectiveDateTime.Time.After(ts)) {
				medicationStatements = append(medicationStatements, ms)
			}
		}
	}
	return medicationStatements, nil
}

func GetPatient(fullFhirUrl string) (*models.Patient, error) {
	resp, err := http.Get(fullFhirUrl)
	defer resp.Body.Close()
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Could not get: %s", fullFhirUrl))
	}
	decoder := json.NewDecoder(resp.Body)
	patient := &models.Patient{}
	err = decoder.Decode(patient)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Could not decode: %s", fullFhirUrl))
	}
	return patient, nil
}
