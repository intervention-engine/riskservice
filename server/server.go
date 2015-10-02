package server

import (
	"time"

	"github.com/intervention-engine/riskservice/assessment"
	"github.com/labstack/echo"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

// Sets up the http request handlers with Echo
func RegisterRiskHandlers(e *echo.Echo, db *mgo.Database, baseUrl string, requestChan chan<- CalculationRequest) {
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
		stringTime := c.Form("timestamp")
		ts, err := time.Parse(time.RFC3339, stringTime)
		if err != nil {
			c.String(400, "Expected timestamp to be populated with an RFC3339 formatted time.")
		}
		requestChan <- CalculationRequest{fhirEndpointUrl, patientId, ts, time.Now()}
		return
	})
}
