package server

import (
	"net/url"
	"strings"
	"time"

	"github.com/intervention-engine/fhir/models"
	"github.com/intervention-engine/riskservice/assessment"
	"github.com/intervention-engine/riskservice/chads"
	"github.com/intervention-engine/riskservice/plugin"
	"github.com/intervention-engine/riskservice/simple"
	"github.com/pebbe/util"
	. "gopkg.in/check.v1"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/dbtest"
)

type ServiceSuite struct {
	DBServer *dbtest.DBServer
	Session  *mgo.Session
	Service  *RiskService
}

var _ = Suite(&ServiceSuite{})

func (s *ServiceSuite) SetUpSuite(c *C) {
	// Set up the database
	s.DBServer = &dbtest.DBServer{}
	s.DBServer.SetPath(c.MkDir())
}

func (s *ServiceSuite) SetUpTest(c *C) {
	s.Session = s.DBServer.Session()
	db := s.Session.DB("riskservice-test")
	s.Service = NewRiskService(db)
}

func (s *ServiceSuite) TearDownTest(c *C) {
	s.Session.Close()
	s.DBServer.Wipe()
}

func (s *ServiceSuite) TearDownSuite(c *C) {
	s.DBServer.Stop()
}

func (s *ServiceSuite) TestSortAndConsolidateOutOfOrder(c *C) {
	one, two, three, four, five := 1, 2, 3, 4, 5
	results := []plugin.RiskServiceCalculationResult{
		{
			AsOf:  time.Date(2012, 1, 1, 11, 0, 0, 0, time.UTC),
			Score: &one,
		}, {
			AsOf:  time.Date(2014, 2, 3, 10, 0, 0, 0, time.UTC),
			Score: &two,
		}, {
			AsOf:  time.Date(2000, 7, 14, 16, 0, 0, 0, time.UTC),
			Score: &three,
		}, {
			AsOf:  time.Date(2013, 1, 1, 11, 0, 0, 0, time.UTC),
			Score: &four,
		}, {
			AsOf:  time.Date(2000, 7, 14, 15, 59, 59, 999, time.UTC),
			Score: &five,
		},
	}

	results = sortAndConsolidate(results)
	c.Assert(results, HasLen, 5)
	c.Assert(results[0].AsOf, Equals, time.Date(2000, 7, 14, 15, 59, 59, 999, time.UTC))
	c.Assert(*results[0].Score, Equals, 5)
	c.Assert(results[1].AsOf, Equals, time.Date(2000, 7, 14, 16, 0, 0, 0, time.UTC))
	c.Assert(*results[1].Score, Equals, 3)
	c.Assert(results[2].AsOf, Equals, time.Date(2012, 1, 1, 11, 0, 0, 0, time.UTC))
	c.Assert(*results[2].Score, Equals, 1)
	c.Assert(results[3].AsOf, Equals, time.Date(2013, 1, 1, 11, 0, 0, 0, time.UTC))
	c.Assert(*results[3].Score, Equals, 4)
	c.Assert(results[4].AsOf, Equals, time.Date(2014, 2, 3, 10, 0, 0, 0, time.UTC))
	c.Assert(*results[4].Score, Equals, 2)
}

func (s *ServiceSuite) TestSortAndConsolidateWithDuplicates(c *C) {
	one, two, three, four, five, six, seven, eight := 1, 2, 3, 4, 5, 6, 7, 8
	results := []plugin.RiskServiceCalculationResult{
		{
			AsOf:  time.Date(2012, 1, 1, 11, 0, 0, 0, time.UTC),
			Score: &one,
		}, {
			AsOf:  time.Date(2014, 2, 3, 10, 0, 0, 0, time.UTC),
			Score: &two,
		}, {
			AsOf:  time.Date(2014, 2, 3, 10, 0, 0, 0, time.UTC),
			Score: &six,
		}, {
			AsOf:  time.Date(2000, 7, 14, 16, 0, 0, 0, time.UTC),
			Score: &three,
		}, {
			AsOf:  time.Date(2000, 7, 14, 16, 0, 0, 0, time.UTC),
			Score: &eight,
		}, {
			AsOf:  time.Date(2000, 7, 14, 16, 0, 0, 0, time.UTC),
			Score: &seven,
		}, {
			AsOf:  time.Date(2013, 1, 1, 11, 0, 0, 0, time.UTC),
			Score: &four,
		}, {
			AsOf:  time.Date(2000, 7, 14, 15, 59, 59, 999, time.UTC),
			Score: &five,
		},
	}

	results = sortAndConsolidate(results)
	c.Assert(results, HasLen, 5)
	c.Assert(results[0].AsOf, Equals, time.Date(2000, 7, 14, 15, 59, 59, 999, time.UTC))
	c.Assert(*results[0].Score, Equals, 5)
	c.Assert(results[1].AsOf, Equals, time.Date(2000, 7, 14, 16, 0, 0, 0, time.UTC))
	c.Assert(*results[1].Score, Equals, 7)
	c.Assert(results[2].AsOf, Equals, time.Date(2012, 1, 1, 11, 0, 0, 0, time.UTC))
	c.Assert(*results[2].Score, Equals, 1)
	c.Assert(results[3].AsOf, Equals, time.Date(2013, 1, 1, 11, 0, 0, 0, time.UTC))
	c.Assert(*results[3].Score, Equals, 4)
	c.Assert(results[4].AsOf, Equals, time.Date(2014, 2, 3, 10, 0, 0, 0, time.UTC))
	c.Assert(*results[4].Score, Equals, 6)
}

func (s *ServiceSuite) TestGetRiskAssessmentDeleteURL(c *C) {
	cc := models.CodeableConcept{
		Coding: []models.Coding{
			{System: "foo", Code: "bar"},
		},
	}
	patientID := "12345"
	delURL := getRiskAssessmentDeleteURL(cc, patientID)
	c.Assert(strings.HasPrefix(delURL, "RiskAssessment?"), Equals, true)
	values, err := url.ParseQuery(strings.TrimPrefix(delURL, "RiskAssessment?"))
	util.CheckErr(err)
	c.Assert(values, HasLen, 2)
	c.Assert(values.Get("patient"), Equals, "12345")
	c.Assert(values.Get("method"), Equals, "foo|bar")
}

func (s *ServiceSuite) TestGetRequiredDataQueryURLForCHADS(c *C) {
	s.Service.RegisterPlugin(chads.NewCHA2DS2VAScPlugin())
	endpoint, _ := url.Parse("http://example.org")
	qURL, err := s.Service.getRequiredDataQueryURL("12345", *endpoint)
	util.CheckErr(err)
	c.Assert(strings.HasPrefix(qURL.String(), "http://example.org/Patient?"), Equals, true)
	c.Assert(qURL.Query(), HasLen, 2)
	c.Assert(qURL.Query().Get("id"), Equals, "12345")
	c.Assert(qURL.Query()["_revinclude"], HasLen, 1)
	c.Assert(qURL.Query().Get("_revinclude"), Equals, "Condition:patient")
}

func (s *ServiceSuite) TestGetRequiredDataQueryURLForSimple(c *C) {
	s.Service.RegisterPlugin(simple.NewSimplePlugin())
	endpoint, _ := url.Parse("http://example.org")
	qURL, err := s.Service.getRequiredDataQueryURL("12345", *endpoint)
	util.CheckErr(err)
	c.Assert(strings.HasPrefix(qURL.String(), "http://example.org/Patient?"), Equals, true)
	c.Assert(qURL.Query(), HasLen, 2)
	c.Assert(qURL.Query().Get("id"), Equals, "12345")
	c.Assert(qURL.Query()["_revinclude"], HasLen, 2)
	var cFound, mFound bool
	for _, v := range qURL.Query()["_revinclude"] {
		if v == "Condition:patient" {
			cFound = true
		} else if v == "MedicationStatement:patient" {
			mFound = true
		}
	}
	c.Assert(cFound, Equals, true)
	c.Assert(mFound, Equals, true)
}

func (s *ServiceSuite) TestGetRequiredDataQueryURLForCHADSandSimple(c *C) {
	s.Service.RegisterPlugin(chads.NewCHA2DS2VAScPlugin())
	s.Service.RegisterPlugin(simple.NewSimplePlugin())
	endpoint, _ := url.Parse("http://example.org")
	qURL, err := s.Service.getRequiredDataQueryURL("12345", *endpoint)
	util.CheckErr(err)
	c.Assert(strings.HasPrefix(qURL.String(), "http://example.org/Patient?"), Equals, true)
	c.Assert(qURL.Query()["_revinclude"], HasLen, 2)
	var cFound, mFound bool
	for _, v := range qURL.Query()["_revinclude"] {
		if v == "Condition:patient" {
			cFound = true
		} else if v == "MedicationStatement:patient" {
			mFound = true
		}
	}
	c.Assert(cFound, Equals, true)
	c.Assert(mFound, Equals, true)
}

func (s *ServiceSuite) TestBuildRiskAssessmentBundle(c *C) {
	one, two, three, four := 1, 2, 3, 4
	results := []plugin.RiskServiceCalculationResult{
		{
			AsOf:  time.Date(2000, 7, 14, 15, 59, 59, 999, time.UTC),
			Score: &one,
			Pie:   assessment.NewPie("Patient/12345"),
		}, {
			AsOf:  time.Date(2000, 7, 14, 16, 0, 0, 0, time.UTC),
			Score: &one,
			Pie:   assessment.NewPie("Patient/12345"),
		}, {
			AsOf:  time.Date(2012, 1, 1, 11, 0, 0, 0, time.UTC),
			Score: &three,
			Pie:   assessment.NewPie("Patient/12345"),
		}, {
			AsOf:  time.Date(2013, 1, 1, 11, 0, 0, 0, time.UTC),
			Score: &four,
			Pie:   assessment.NewPie("Patient/12345"),
		}, {
			AsOf:  time.Date(2014, 2, 3, 10, 0, 0, 0, time.UTC),
			Score: &two,
			Pie:   assessment.NewPie("Patient/12345"),
		},
	}
	pieBasisURL, _ := url.Parse("http://example.org/Pie")
	simplePlugin := simple.NewSimplePlugin()
	bundle := buildRiskAssessmentBundle("12345", results, *pieBasisURL, simplePlugin)

	c.Assert(bundle, NotNil)
	c.Assert(bundle.Type, Equals, "transaction")
	c.Assert(bundle.Entry, HasLen, 6)

	// First check the delete entry
	c.Assert(bundle.Entry[0].FullUrl, Equals, "")
	c.Assert(bundle.Entry[0].Link, HasLen, 0)
	c.Assert(bundle.Entry[0].Resource, IsNil)
	c.Assert(bundle.Entry[0].Response, IsNil)
	c.Assert(bundle.Entry[0].Search, IsNil)
	c.Assert(bundle.Entry[0].Request.Method, Equals, "DELETE")
	delURL := bundle.Entry[0].Request.Url
	c.Assert(strings.HasPrefix(delURL, "RiskAssessment?"), Equals, true)
	delValues, err := url.ParseQuery(strings.TrimPrefix(delURL, "RiskAssessment?"))
	util.CheckErr(err)
	c.Assert(delValues, HasLen, 2)
	c.Assert(delValues.Get("patient"), Equals, "12345")
	c.Assert(delValues.Get("method"), Equals, "http://interventionengine.org/risk-assessments|Simple")

	// Now check the post entries
	for i := 1; i < len(bundle.Entry); i++ {
		entry := bundle.Entry[i]
		result := results[i-1]
		c.Assert(entry.FullUrl, Equals, "")
		c.Assert(entry.Link, HasLen, 0)
		c.Assert(entry.Response, IsNil)
		c.Assert(entry.Search, IsNil)
		c.Assert(entry.Request.Method, Equals, "POST")
		c.Assert(entry.Request.Url, Equals, "RiskAssessment")
		c.Assert(entry.Resource, NotNil)
		ra, ok := entry.Resource.(*models.RiskAssessment)
		c.Assert(ok, Equals, true)
		c.Assert(ra.Id, Equals, "")
		c.Assert(ra.Basis, HasLen, 1)
		c.Assert(ra.Basis[0].Reference, Equals, pieBasisURL.String()+"/"+result.Pie.Id.Hex())
		c.Assert(ra.Date.Time, DeepEquals, result.AsOf)
		c.Assert(ra.Date.Precision, Equals, models.Precision(models.Timestamp))
		c.Assert(*ra.Method, DeepEquals, simplePlugin.Config().Method)
		c.Assert(ra.Prediction, HasLen, 1)
		c.Assert(*ra.Prediction[0].Outcome, DeepEquals, simplePlugin.Config().PredictedOutcome)
		c.Assert(*ra.Prediction[0].ProbabilityDecimal, Equals, float64(*result.Score))
		c.Assert(ra.Subject.Reference, Equals, "Patient/12345")
	}
}
