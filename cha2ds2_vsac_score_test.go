package riskservice

import (
	"encoding/json"
	"github.com/intervention-engine/fhir/models"
	"github.com/pebbe/util"
	. "gopkg.in/check.v1"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"time"
)

type CHADSSuite struct {
	Bundle *models.ConditionBundle
	Server *httptest.Server
}

var _ = Suite(&CHADSSuite{})

func (cs *CHADSSuite) SetUpSuite(c *C) {
	data, err := os.Open("fixtures/condition_bundle.json")
	defer data.Close()
	util.CheckErr(err)
	decoder := json.NewDecoder(data)
	bundle := &models.ConditionBundle{}
	err = decoder.Decode(bundle)
	util.CheckErr(err)
	cs.Bundle = bundle

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.RequestURI, "Condition") {
			json.NewEncoder(w).Encode(bundle)
		}
		if strings.Contains(r.RequestURI, "Patient") {
			birthDate := &models.FHIRDateTime{Time: time.Date(1945, time.July, 1, 0, 0, 0, 0, time.UTC), Precision: models.Date}
			patient := &models.Patient{BirthDate: birthDate, Gender: "female"}
			json.NewEncoder(w).Encode(patient)
		}
	})
	cs.Server = httptest.NewServer(handler)
}

func (cs *CHADSSuite) TearDownSuite(c *C) {
	cs.Server.Close()
}

func (cs *CHADSSuite) TestAge(c *C) {
	birthDate := &models.FHIRDateTime{Time: time.Date(1978, time.July, 1, 0, 0, 0, 0, time.UTC), Precision: models.Date}
	patient := &models.Patient{BirthDate: birthDate}
	age := Age(patient)
	c.Assert(age, Equals, 37)
}

func (cs *CHADSSuite) TestFuzzyFindConditionInBundle(c *C) {
	c.Assert(FuzzyFindConditionInBundle("401", "http://hl7.org/fhir/sid/icd-9", cs.Bundle), Equals, true)
	c.Assert(FuzzyFindConditionInBundle("500", "http://hl7.org/fhir/sid/icd-9", cs.Bundle), Equals, false)
}

func (cs *CHADSSuite) TestCalculateConditionPortion(c *C) {
	pie := NewPie("")
	c.Assert(CalculateConditionPortion(cs.Bundle, pie), Equals, 3)
	c.Assert(len(pie.Slices), Equals, 5)
	c.Assert(pie.Slices[0].Name, Equals, "Congestive Heart Failure")
	c.Assert(pie.Slices[0].Weight, Equals, PieSliceWidth)
	c.Assert(pie.Slices[0].Value, Equals, 0)
	c.Assert(pie.Slices[1].Name, Equals, "Hypertension")
	c.Assert(pie.Slices[1].Weight, Equals, PieSliceWidth)
	c.Assert(pie.Slices[1].Value, Equals, 1)
	c.Assert(pie.Slices[3].Name, Equals, "Stroke")
	c.Assert(pie.Slices[3].Weight, Equals, PieSliceWidth*2)
	c.Assert(pie.Slices[3].Value, Equals, 2)
}

func (cs *CHADSSuite) TestCalculateDemographicPortion(c *C) {
	pie := NewPie("")
	birthDate := &models.FHIRDateTime{Time: time.Date(1945, time.July, 1, 0, 0, 0, 0, time.UTC), Precision: models.Date}
	patient := &models.Patient{BirthDate: birthDate, Gender: "female"}
	c.Assert(CalculateDemographicPortion(patient, pie), Equals, 2)
	c.Assert(len(pie.Slices), Equals, 2)
	c.Assert(pie.Slices[0].Name, Equals, "Gender")
	c.Assert(pie.Slices[0].Weight, Equals, PieSliceWidth)
	c.Assert(pie.Slices[0].Value, Equals, 1)
	c.Assert(pie.Slices[1].Name, Equals, "Age")
	c.Assert(pie.Slices[1].Weight, Equals, PieSliceWidth*2)
	c.Assert(pie.Slices[1].Value, Equals, 1)
}

func (cs *CHADSSuite) TestCalculateCHADSRisk(c *C) {
	ra, pie, err := CalculateCHADSRisk(cs.Server.URL, "foo")
	util.CheckErr(err)
	c.Assert(*ra.Prediction[0].ProbabilityDecimal, Equals, 6.7)
	c.Assert(len(pie.Slices), Equals, 7)
}
