package fhir

import (
	"fmt"
)

func PatientUrl(fhirEndpointUrl, patientId string) string {
	return fmt.Sprintf("%s/Patient/%s", fhirEndpointUrl, patientId)
}

func ResourcesForPatientUrl(fhirEndpointUrl, patientId, resource string) string {
	return fmt.Sprintf("%s/%s?patient:Patient=%s", fhirEndpointUrl, resource, patientId)
}

func ResourcesForPatientByCodeUrl(fhirEndpointUrl, patientId, resource, code, system string) string {
	url := ResourcesForPatientUrl(fhirEndpointUrl, patientId, resource)
	return fmt.Sprintf("%s&code=%s|%s", url, system, code)
}
