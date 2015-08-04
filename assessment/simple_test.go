package assessment

import (
	"bytes"
	"github.com/pebbe/util"
	. "gopkg.in/check.v1"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

type SimpleSuite struct {
	Server *httptest.Server
}

func Test(t *testing.T) { TestingT(t) }

var _ = Suite(&SimpleSuite{})

func (s *SimpleSuite) SetUpSuite(c *C) {
	data, err := os.Open("fixtures/condition_bundle.json")
	util.CheckErr(err)
	defer data.Close()
	buf := new(bytes.Buffer)
	buf.ReadFrom(data)
	jsonString := buf.String()

	util.CheckErr(err)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.RequestURI, "Condition") {
			jr := strings.NewReader(jsonString)
			jr.WriteTo(w)
		}
		if strings.Contains(r.RequestURI, "MedicationStatement") {
			w.Write([]byte("{\"total\": 0}"))
		}
	})
	s.Server = httptest.NewServer(handler)
}

func (s *SimpleSuite) TearDownSuite(c *C) {
	s.Server.Close()
}

func (s *SimpleSuite) TestCalculateSimpleRisk(c *C) {
	assessment, pie, err := CalculateSimpleRisk(s.Server.URL, "5", time.Date(2015, time.August, 1, 0, 0, 0, 0, time.UTC))
	util.CheckErr(err)
	c.Assert(*assessment.Prediction[0].ProbabilityDecimal, Equals, float64(3))
	c.Assert(pie.Slices[0].Name, Equals, "Conditions")
	c.Assert(pie.Slices[0].Value, Equals, 3)
	c.Assert(pie.Slices[1].Name, Equals, "Medications")
	c.Assert(pie.Slices[1].Value, Equals, 0)
}
