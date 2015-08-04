package server

import (
	"github.com/intervention-engine/fhir/models"
	"github.com/intervention-engine/fhir/upload"
	"github.com/intervention-engine/riskservice/assessment"
	"gopkg.in/mgo.v2"
	"time"
)

type RiskAssessmentCalculation func(fhirEndpointUrl, patientId string, ts time.Time) (*models.RiskAssessment, *assessment.Pie, error)

func CreateRiskAssessment(fhirEndpointUrl, patientId, basePieUrl string, rac RiskAssessmentCalculation, db *mgo.Database, ts time.Time) error {
	ra, pie, err := rac(fhirEndpointUrl, patientId, ts)
	if err != nil {
		return err
	}
	pieCollection := db.C("pies")
	err = pieCollection.Insert(pie)
	if err != nil {
		return err
	}
	ra.Basis = []models.Reference{models.Reference{Reference: basePieUrl + pie.Id.Hex()}}
	_, err = upload.UploadResource(ra, fhirEndpointUrl)
	if err != nil {
		return err
	}

	return nil
}
