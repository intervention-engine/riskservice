package assessment

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"time"

	"github.com/intervention-engine/fhir/models"
	"github.com/pebbe/util"
	. "gopkg.in/check.v1"
)

type CHADSSuite struct {
	Bundle     *models.Bundle
	Server     *httptest.Server
	Conditions []*models.Condition
}

var _ = Suite(&CHADSSuite{})

func (cs *CHADSSuite) SetUpSuite(c *C) {
	data, err := os.Open("fixtures/condition_bundle.json")
	util.CheckErr(err)
	defer data.Close()
	buf := new(bytes.Buffer)
	buf.ReadFrom(data)
	jsonString := buf.String()
	decoder := json.NewDecoder(strings.NewReader(jsonString))
	bundle := &models.Bundle{}
	err = decoder.Decode(bundle)
	util.CheckErr(err)
	cs.Bundle = bundle

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.RequestURI, "Condition") {
			jr := strings.NewReader(jsonString)
			jr.WriteTo(w)
		}
		if strings.Contains(r.RequestURI, "Patient") {
			birthDate := &models.FHIRDateTime{Time: time.Date(1945, time.July, 1, 0, 0, 0, 0, time.UTC), Precision: models.Date}
			patient := &models.Patient{BirthDate: birthDate, Gender: "female"}
			json.NewEncoder(w).Encode(patient)
		}
	})
	cs.Server = httptest.NewServer(handler)

	var conditions []*models.Condition
	for _, resource := range bundle.Entry {
		c, ok := resource.Resource.(*models.Condition)
		if ok {
			conditions = append(conditions, c)
		}
	}

	cs.Conditions = conditions
}

func (cs *CHADSSuite) TearDownSuite(c *C) {
	cs.Server.Close()
}

func (cs *CHADSSuite) TestAge(c *C) {
	birthDate := &models.FHIRDateTime{Time: time.Date(1978, time.July, 1, 0, 0, 0, 0, time.UTC), Precision: models.Date}
	patient := &models.Patient{BirthDate: birthDate}
	age := Age(patient, time.Date(2015, time.August, 1, 0, 0, 0, 0, time.UTC))
	c.Assert(age, Equals, 37)
}

func (cs *CHADSSuite) TestFuzzyFindInConditions(c *C) {

	c.Assert(FuzzyFindInConditions("401", "http://hl7.org/fhir/sid/icd-9", cs.Conditions), Equals, true)
	c.Assert(FuzzyFindInConditions("500", "http://hl7.org/fhir/sid/icd-9", cs.Conditions), Equals, false)
}

func (cs *CHADSSuite) TestCalculateConditionPortion(c *C) {
	pie := NewPie("")
	c.Assert(CalculateConditionPortion(cs.Conditions, pie), Equals, 3)
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
	c.Assert(CalculateDemographicPortion(patient, pie, time.Date(2015, time.August, 1, 0, 0, 0, 0, time.UTC)), Equals, 2)
	c.Assert(len(pie.Slices), Equals, 2)
	c.Assert(pie.Slices[0].Name, Equals, "Gender")
	c.Assert(pie.Slices[0].Weight, Equals, PieSliceWidth)
	c.Assert(pie.Slices[0].Value, Equals, 1)
	c.Assert(pie.Slices[1].Name, Equals, "Age")
	c.Assert(pie.Slices[1].Weight, Equals, PieSliceWidth*2)
	c.Assert(pie.Slices[1].Value, Equals, 1)
}

func (cs *CHADSSuite) TestCalculateCHADSRisk(c *C) {
	ra, pie, err := CalculateCHADSRisk(cs.Server.URL, "foo", time.Date(2015, time.August, 1, 0, 0, 0, 0, time.UTC))
	util.CheckErr(err)
	c.Assert(*ra.Prediction[0].ProbabilityDecimal, Equals, 6.7)
	c.Assert(len(pie.Slices), Equals, 7)
}
