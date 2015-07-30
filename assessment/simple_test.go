package assessment

import (
	"github.com/pebbe/util"
	. "gopkg.in/check.v1"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type SimpleSuite struct {
	Server *httptest.Server
}

func Test(t *testing.T) { TestingT(t) }

var _ = Suite(&SimpleSuite{})

func (s *SimpleSuite) SetUpSuite(c *C) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.RequestURI, "Condition") {
			w.Write([]byte("{\"total\": 5}"))
		}
		if strings.Contains(r.RequestURI, "MedicationStatement") {
			w.Write([]byte("{\"total\": 1}"))
		}
	})
	s.Server = httptest.NewServer(handler)
}

func (s *SimpleSuite) TearDownSuite(c *C) {
	s.Server.Close()
}

func (s *SimpleSuite) TestCalculateSimpleRisk(c *C) {
	assessment, pie, err := CalculateSimpleRisk(s.Server.URL, "5")
	util.CheckErr(err)
	c.Assert(*assessment.Prediction[0].ProbabilityDecimal, Equals, float64(6))
	c.Assert(pie.Slices[0].Name, Equals, "Conditions")
	c.Assert(pie.Slices[0].Value, Equals, 5)
	c.Assert(pie.Slices[1].Name, Equals, "MedicationStatement")
	c.Assert(pie.Slices[1].Value, Equals, 1)
}
