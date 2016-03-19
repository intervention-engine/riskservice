package server

import (
	"fmt"

	"github.com/intervention-engine/riskservice/assessment"
	"github.com/labstack/echo"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

// RegisterRoutes sets up the http request handlers with Echo
func RegisterRoutes(e *echo.Echo, db *mgo.Database, basePieURL string, service RiskService, fnDelayer *FunctionDelayer) {
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
		patientID := c.Form("patientId")
		fhirEndpointURL := c.Form("fhirEndpointUrl")
		key := fmt.Sprintf("%s@%s", patientID, fhirEndpointURL)
		fnDelayer.Delay(key, func() {
			service.Calculate(patientID, fhirEndpointURL, basePieURL)
		})
		return
	})
}
