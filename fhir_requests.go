package riskservice

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/intervention-engine/fhir/models"
	"net/http"
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

func GetPatientConditions(fullFhirUrl string) (*models.ConditionBundle, error) {
	resp, err := http.Get(fullFhirUrl)
	defer resp.Body.Close()
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Could not get: %s", fullFhirUrl))
	}
	decoder := json.NewDecoder(resp.Body)
	bundle := &models.ConditionBundle{}
	err = decoder.Decode(bundle)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Could not decode: %s", fullFhirUrl))
	}
	return bundle, nil
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
