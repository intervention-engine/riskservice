package server

import (
	"github.com/intervention-engine/riskservice/assessment"
	"github.com/labstack/echo"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

func RegisterRiskHandlers(e *echo.Echo, db *mgo.Database, baseUrl string) {
	e.Get("/pies/:id", func(c *echo.Context) (err error) {
		pie := &assessment.Pie{}
		id := c.Param("id")
		if bson.IsObjectIdHex(id) {
			query := db.C("pies").FindId(bson.ObjectIdHex(id))
			err = query.One(pie)
			if err == nil {
				c.JSON(200, pie)
			}
		} else {
			c.String(400, "Bad ID format for requested Pie. Should be a BSON Id")
		}
		return
	})

	e.Post("/calculate", func(c *echo.Context) (err error) {
		patientId := c.Form("patientId")
		fhirEndpointUrl := c.Form("fhirEndpointUrl")
		riskAssessments := []RiskAssessmentCalculation{assessment.CalculateCHADSRisk, assessment.CalculateSimpleRisk}
		for _, rac := range riskAssessments {
			err = CreateRiskAssessment(fhirEndpointUrl, patientId, baseUrl, rac, db)
			if err != nil {
				return
			}
		}
		return
	})
}
