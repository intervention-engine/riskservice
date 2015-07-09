package riskservice

import (
	"github.com/pebbe/util"
	. "gopkg.in/check.v1"
	"net/http"
	"net/http/httptest"
	"strings"
)

type SimpleSuite struct {
	Server *httptest.Server
}

var _ = Suite(&SimpleSuite{})

func (s *SimpleSuite) SetUpSuite(c *C) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.RequestURI, "Condition") {
			w.Write([]byte("{'Total': 5}"))
		}
		if strings.Contains(r.RequestURI, "MedicationStatement") {
			w.Write([]byte("{'Total': 1}"))
		}
	})
	s.Server = httptest.NewServer(handler)
	s.Server.Start()
}

func (s *SimpleSuite) TearDownSuite(c *C) {
	s.Server.Close()
}

func (s *SimpleSuite) TestCalculateSimpleRisk(c *C) {
	assessment, err := CalculateSimpleRisk(s.Server.URL, "5")
	util.CheckErr(err)
	c.Assert(assessment.Prediction[0].ProbabilityDecimal, Equals, 6)
}
