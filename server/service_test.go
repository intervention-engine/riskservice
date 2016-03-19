package server

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/intervention-engine/fhir/models"
	"github.com/intervention-engine/fhir/server"
	"github.com/intervention-engine/riskservice/assessment"
	"github.com/intervention-engine/riskservice/chads"
	"github.com/intervention-engine/riskservice/plugin"
	"github.com/intervention-engine/riskservice/simple"
	"github.com/labstack/echo"
	"github.com/pebbe/util"
	. "gopkg.in/check.v1"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"gopkg.in/mgo.v2/dbtest"
)

type ServiceSuite struct {
	DBServer *dbtest.DBServer
	Session  *mgo.Session
	Database *mgo.Database
	Service  *ReferenceRiskService
	Server   *httptest.Server
}

var _ = Suite(&ServiceSuite{})

func (s *ServiceSuite) SetUpSuite(c *C) {
	// Set up the database
	s.DBServer = &dbtest.DBServer{}
	s.DBServer.SetPath(c.MkDir())
}

func (s *ServiceSuite) SetUpTest(c *C) {
	s.Session = s.DBServer.Session()
	s.Database = s.Session.DB("riskservice-test")
	s.Service = NewReferenceRiskService(s.Database)

	e := echo.New()
	server.RegisterRoutes(e, make(map[string][]echo.Middleware), server.NewMongoDataAccessLayer(s.Database), server.Config{})
	s.Server = httptest.NewServer(e.Router())
}

func (s *ServiceSuite) TearDownTest(c *C) {
	s.Server.Close()
	s.Session.Close()
	s.DBServer.Wipe()
}

func (s *ServiceSuite) TearDownSuite(c *C) {
	s.DBServer.Stop()
}

func (s *ServiceSuite) TestEndToEndCalculations(c *C) {
	// Get the test data for Chad Chadworth (for the CHADS test -- get it?)
	data, err := os.Open("fixtures/chad_chadworth_bundle.json")
	util.CheckErr(err)
	defer data.Close()

	// Store Chad Chadworth's information to Mongo
	res, err := http.Post(s.Server.URL+"/", "application/json", data)
	util.CheckErr(err)
	defer res.Body.Close()

	// Get the response so we can pull out the patient ID
	decoder := json.NewDecoder(res.Body)
	responseBundle := new(models.Bundle)
	err = decoder.Decode(responseBundle)
	util.CheckErr(err)
	patientID := responseBundle.Entry[0].Resource.(*models.Patient).Id

	// Confirm there are no risk assessments or pies
	raCollection := s.Database.C("riskassessments")
	count, err := raCollection.Count()
	util.CheckErr(err)
	c.Assert(count, Equals, 0)
	piesCollection := s.Database.C("pies")
	count, err = piesCollection.Count()
	util.CheckErr(err)
	c.Assert(count, Equals, 0)

	// Now register the plugins and request the calculation!
	s.Service.RegisterPlugin(chads.NewCHA2DS2VAScPlugin())
	s.Service.RegisterPlugin(simple.NewSimplePlugin())

	err = s.Service.Calculate(patientID, s.Server.URL, s.Server.URL+"/pies")
	util.CheckErr(err)

	count, err = raCollection.Find(bson.M{"method.coding.code": "CHADS"}).Count()
	util.CheckErr(err)
	c.Assert(count, Equals, 4)
	count, err = raCollection.Find(bson.M{"method.coding.code": "Simple"}).Count()
	util.CheckErr(err)
	c.Assert(count, Equals, 4)
	count, err = piesCollection.Count()
	util.CheckErr(err)
	c.Assert(count, Equals, 8)

	var ras []models.RiskAssessment
	err = raCollection.Find(bson.M{"method.coding.code": "CHADS"}).Sort("date.time").All(&ras)
	util.CheckErr(err)

	loc := time.FixedZone("-0500", -5*60*60)
	s.checkCHADSRiskAssessment(c, &ras[0], patientID, time.Date(2012, time.September, 20, 8, 0, 0, 0, loc), 1.3, false)
	s.checkCHADSPie(c, &ras[0], patientID, 0, 0, 0, 0, 0, 1, 0)
	s.checkCHADSRiskAssessment(c, &ras[1], patientID, time.Date(2013, time.September, 2, 10, 0, 0, 0, loc), 2.2, false)
	s.checkCHADSPie(c, &ras[1], patientID, 0, 1, 0, 0, 0, 1, 0)
	s.checkCHADSRiskAssessment(c, &ras[2], patientID, time.Date(2014, time.January, 17, 20, 35, 0, 0, loc), 4.0, false)
	s.checkCHADSPie(c, &ras[2], patientID, 0, 1, 0, 2, 0, 1, 0)
	s.checkCHADSRiskAssessment(c, &ras[3], patientID, time.Date(2015, time.September, 2, 0, 0, 0, 0, loc), 6.7, true)
	s.checkCHADSPie(c, &ras[3], patientID, 0, 1, 0, 2, 0, 2, 0)

	ras = nil
	err = raCollection.Find(bson.M{"method.coding.code": "Simple"}).Sort("date.time").All(&ras)
	util.CheckErr(err)

	s.checkSimpleRiskAssessment(c, &ras[0], patientID, time.Date(2012, time.September, 20, 8, 0, 0, 0, loc), 1, false)
	s.checkSimplePie(c, &ras[0], patientID, 1, 0)
	s.checkSimpleRiskAssessment(c, &ras[1], patientID, time.Date(2013, time.September, 2, 10, 0, 0, 0, loc), 3, false)
	s.checkSimplePie(c, &ras[1], patientID, 2, 1)
	s.checkSimpleRiskAssessment(c, &ras[2], patientID, time.Date(2014, time.January, 17, 20, 35, 0, 0, loc), 4, false)
	s.checkSimplePie(c, &ras[2], patientID, 3, 1)
	s.checkSimpleRiskAssessment(c, &ras[3], patientID, time.Date(2014, time.January, 17, 20, 40, 0, 0, loc), 3, true)
	s.checkSimplePie(c, &ras[3], patientID, 2, 1)
}

func (s *ServiceSuite) TestEndToEndOverwritingCalculations(c *C) {
	// We should be able to run the end-to-end test multiple times with the same results.
	// In other words, the risk assessments and pies should not build up.

	// Get the test data for Chad Chadworth (for the CHADS test -- get it?)
	data, err := os.Open("fixtures/chad_chadworth_bundle.json")
	util.CheckErr(err)
	defer data.Close()

	// Store Chad Chadworth's information to Mongo
	res, err := http.Post(s.Server.URL+"/", "application/json", data)
	util.CheckErr(err)
	defer res.Body.Close()

	// Get the response so we can pull out the patient ID
	decoder := json.NewDecoder(res.Body)
	responseBundle := new(models.Bundle)
	err = decoder.Decode(responseBundle)
	util.CheckErr(err)
	patientID := responseBundle.Entry[0].Resource.(*models.Patient).Id

	// Confirm there are no risk assessments or pies
	raCollection := s.Database.C("riskassessments")
	count, err := raCollection.Count()
	util.CheckErr(err)
	c.Assert(count, Equals, 0)
	piesCollection := s.Database.C("pies")
	count, err = piesCollection.Count()
	util.CheckErr(err)
	c.Assert(count, Equals, 0)

	// Now register the plugins and request the calculation!
	s.Service.RegisterPlugin(chads.NewCHA2DS2VAScPlugin())
	s.Service.RegisterPlugin(simple.NewSimplePlugin())

	// This is where we run it a bunch of times, but since it *should* clean up old data every time,
	// then it should have the same results as a single time.
	for i := 0; i < 5; i++ {
		err = s.Service.Calculate(patientID, s.Server.URL, s.Server.URL+"/pies")
		util.CheckErr(err)

		count, err = raCollection.Find(bson.M{"method.coding.code": "CHADS"}).Count()
		util.CheckErr(err)
		c.Assert(count, Equals, 4)
		count, err = raCollection.Find(bson.M{"method.coding.code": "Simple"}).Count()
		util.CheckErr(err)
		c.Assert(count, Equals, 4)
		count, err = piesCollection.Count()
		util.CheckErr(err)
		c.Assert(count, Equals, 8)

		var ras []models.RiskAssessment
		err = raCollection.Find(bson.M{"method.coding.code": "CHADS"}).Sort("date.time").All(&ras)
		util.CheckErr(err)

		loc := time.FixedZone("-0500", -5*60*60)
		s.checkCHADSRiskAssessment(c, &ras[0], patientID, time.Date(2012, time.September, 20, 8, 0, 0, 0, loc), 1.3, false)
		s.checkCHADSPie(c, &ras[0], patientID, 0, 0, 0, 0, 0, 1, 0)
		s.checkCHADSRiskAssessment(c, &ras[1], patientID, time.Date(2013, time.September, 2, 10, 0, 0, 0, loc), 2.2, false)
		s.checkCHADSPie(c, &ras[1], patientID, 0, 1, 0, 0, 0, 1, 0)
		s.checkCHADSRiskAssessment(c, &ras[2], patientID, time.Date(2014, time.January, 17, 20, 35, 0, 0, loc), 4.0, false)
		s.checkCHADSPie(c, &ras[2], patientID, 0, 1, 0, 2, 0, 1, 0)
		s.checkCHADSRiskAssessment(c, &ras[3], patientID, time.Date(2015, time.September, 2, 0, 0, 0, 0, loc), 6.7, true)
		s.checkCHADSPie(c, &ras[3], patientID, 0, 1, 0, 2, 0, 2, 0)

		ras = nil
		err = raCollection.Find(bson.M{"method.coding.code": "Simple"}).Sort("date.time").All(&ras)
		util.CheckErr(err)

		s.checkSimpleRiskAssessment(c, &ras[0], patientID, time.Date(2012, time.September, 20, 8, 0, 0, 0, loc), 1, false)
		s.checkSimplePie(c, &ras[0], patientID, 1, 0)
		s.checkSimpleRiskAssessment(c, &ras[1], patientID, time.Date(2013, time.September, 2, 10, 0, 0, 0, loc), 3, false)
		s.checkSimplePie(c, &ras[1], patientID, 2, 1)
		s.checkSimpleRiskAssessment(c, &ras[2], patientID, time.Date(2014, time.January, 17, 20, 35, 0, 0, loc), 4, false)
		s.checkSimplePie(c, &ras[2], patientID, 3, 1)
		s.checkSimpleRiskAssessment(c, &ras[3], patientID, time.Date(2014, time.January, 17, 20, 40, 0, 0, loc), 3, true)
		s.checkSimplePie(c, &ras[3], patientID, 2, 1)
	}
}

func (s *ServiceSuite) checkCHADSRiskAssessment(c *C, ra *models.RiskAssessment, patientID string, date time.Time, probability float64, mostRecent bool) {
	c.Assert(ra.Subject.Reference, Equals, "Patient/"+patientID)
	c.Assert(ra.Method.MatchesCode("http://interventionengine.org/risk-assessments", "CHADS"), Equals, true)
	c.Assert(ra.Date.Time.Equal(date), Equals, true)
	c.Assert(ra.Prediction, HasLen, 1)
	c.Assert(ra.Prediction[0].Outcome.Text, Equals, "Stroke")
	c.Assert(*ra.Prediction[0].ProbabilityDecimal, Equals, probability)
	c.Assert(ra.Basis, HasLen, 1)
	c.Assert(strings.HasPrefix(ra.Basis[0].Reference, s.Server.URL+"/pies/"), Equals, true)
	if mostRecent {
		c.Assert(ra.Meta.Tag, HasLen, 1)
		c.Assert(ra.Meta.Tag[0], DeepEquals, models.Coding{System: "http://interventionengine.org/tags/", Code: "MOST_RECENT"})
	} else {
		c.Assert(ra.Meta.Tag, HasLen, 0)
	}
}

func (s *ServiceSuite) checkCHADSPie(c *C, ra *models.RiskAssessment, patientID string, chf, hypertension, diabetes, stroke, vascular, age, gender int) {
	pieID := strings.TrimPrefix(ra.Basis[0].Reference, s.Server.URL+"/pies/")
	pie := new(assessment.Pie)
	err := s.Database.C("pies").FindId(bson.ObjectIdHex(pieID)).One(pie)
	util.CheckErr(err)
	// TODO: This should really be full patient URL
	c.Assert(pie.Patient, Equals, s.Server.URL+"/Patient/"+patientID)
	c.Assert(pie.Slices, HasLen, 7)
	c.Assert(pie.Slices[0].Value, Equals, chf)
	c.Assert(pie.Slices[1].Value, Equals, hypertension)
	c.Assert(pie.Slices[2].Value, Equals, diabetes)
	c.Assert(pie.Slices[3].Value, Equals, stroke)
	c.Assert(pie.Slices[4].Value, Equals, vascular)
	c.Assert(pie.Slices[5].Value, Equals, age)
	c.Assert(pie.Slices[6].Value, Equals, gender)
}

func (s *ServiceSuite) checkSimpleRiskAssessment(c *C, ra *models.RiskAssessment, patientID string, date time.Time, probability float64, mostRecent bool) {
	c.Assert(ra.Subject.Reference, Equals, "Patient/"+patientID)
	c.Assert(ra.Method.MatchesCode("http://interventionengine.org/risk-assessments", "Simple"), Equals, true)
	c.Assert(ra.Date.Time.Equal(date), Equals, true)
	c.Assert(ra.Prediction, HasLen, 1)
	c.Assert(ra.Prediction[0].Outcome.Text, Equals, "Negative Outcome")
	c.Assert(*ra.Prediction[0].ProbabilityDecimal, Equals, probability)
	c.Assert(ra.Basis, HasLen, 1)
	c.Assert(strings.HasPrefix(ra.Basis[0].Reference, s.Server.URL+"/pies/"), Equals, true)
	if mostRecent {
		c.Assert(ra.Meta.Tag, HasLen, 1)
		c.Assert(ra.Meta.Tag[0], DeepEquals, models.Coding{System: "http://interventionengine.org/tags/", Code: "MOST_RECENT"})
	} else {
		c.Assert(ra.Meta.Tag, HasLen, 0)
	}
}

func (s *ServiceSuite) checkSimplePie(c *C, ra *models.RiskAssessment, patientID string, conditions, medications int) {
	pieID := strings.TrimPrefix(ra.Basis[0].Reference, s.Server.URL+"/pies/")
	pie := new(assessment.Pie)
	err := s.Database.C("pies").FindId(bson.ObjectIdHex(pieID)).One(pie)
	util.CheckErr(err)
	// TODO: This should really be full patient URL
	c.Assert(pie.Patient, Equals, s.Server.URL+"/Patient/"+patientID)
	c.Assert(pie.Slices, HasLen, 2)
	c.Assert(pie.Slices[0].Value, Equals, conditions)
	c.Assert(pie.Slices[1].Value, Equals, medications)
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

func (s *ServiceSuite) TestBundleToEventStreamUnsupportedEvent(c *C) {
	data, err := ioutil.ReadFile("fixtures/brad_bradworth_event_source_bundle.json")
	util.CheckErr(err)

	bundle := new(models.Bundle)
	json.Unmarshal(data, bundle)

	bundle.Entry = append(bundle.Entry, models.BundleEntryComponent{
		Resource: &models.Encounter{},
		Search: &models.BundleEntrySearchComponent{
			Mode: "include",
		},
	})

	_, err = BundleToEventStream(bundle)
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Equals, "Unsupported: Converting Encounter to Event")
}

func (s *ServiceSuite) TestBundleToEventStream(c *C) {
	data, err := ioutil.ReadFile("fixtures/brad_bradworth_event_source_bundle.json")
	util.CheckErr(err)

	bundle := new(models.Bundle)
	json.Unmarshal(data, bundle)

	es, err := BundleToEventStream(bundle)
	util.CheckErr(err)

	c.Assert(es.Patient, NotNil)
	c.Assert(es.Patient.Id, Equals, "507f1f77bcf86cd799439001")
	c.Assert(es.Events, HasLen, 5)
	loc := time.FixedZone("-0500", -5*60*60)
	// Event 0 (Condition: Atrial Fibrillation)
	c.Assert(es.Events[0].Date.Equal(time.Date(2012, time.September, 20, 8, 0, 0, 0, loc)), Equals, true)
	c.Assert(es.Events[0].Type, Equals, "Condition")
	c.Assert(es.Events[0].End, Equals, false)
	c.Assert(es.Events[0].Value, DeepEquals, bundle.Entry[1].Resource)
	// Event 1 (Condition: Hypertension)
	c.Assert(es.Events[1].Date.Equal(time.Date(2013, time.September, 2, 10, 0, 0, 0, loc)), Equals, true)
	c.Assert(es.Events[1].Type, Equals, "Condition")
	c.Assert(es.Events[1].End, Equals, false)
	c.Assert(es.Events[1].Value, DeepEquals, bundle.Entry[2].Resource)
	// Event 2 (MedicationStatement: Lisinopril)
	c.Assert(es.Events[2].Date.Equal(time.Date(2013, time.September, 2, 10, 0, 0, 0, loc)), Equals, true)
	c.Assert(es.Events[2].Type, Equals, "MedicationStatement")
	c.Assert(es.Events[2].End, Equals, false)
	c.Assert(es.Events[2].Value, DeepEquals, bundle.Entry[4].Resource)
	// Event 3 (Condition: Cerebral infarction due to cerebral artery occlusion)
	c.Assert(es.Events[3].Date.Equal(time.Date(2014, time.January, 17, 20, 35, 0, 0, loc)), Equals, true)
	c.Assert(es.Events[3].Type, Equals, "Condition")
	c.Assert(es.Events[3].End, Equals, false)
	c.Assert(es.Events[3].Value, DeepEquals, bundle.Entry[3].Resource)
	// Event 4 (Condition END: Cerebral infarction due to cerebral artery occlusion)
	c.Assert(es.Events[4].Date.Equal(time.Date(2014, time.January, 17, 20, 40, 0, 0, loc)), Equals, true)
	c.Assert(es.Events[4].Type, Equals, "Condition")
	c.Assert(es.Events[4].End, Equals, true)
	c.Assert(es.Events[4].Value, DeepEquals, bundle.Entry[3].Resource)
}

func (s *ServiceSuite) TestAddSignificantBirthdays(c *C) {
	bd := time.Date(1950, time.March, 1, 12, 0, 0, 0, time.UTC)
	es := &plugin.EventStream{
		Patient: &models.Patient{
			BirthDate: &models.FHIRDateTime{Time: bd, Precision: models.Precision(models.Timestamp)},
		},
		Events: []plugin.Event{
			{
				Date:  time.Date(1985, time.January, 1, 12, 0, 0, 0, time.UTC),
				Type:  "Condition",
				End:   false,
				Value: new(models.Condition),
			},
			{
				Date:  time.Date(2010, time.February, 1, 12, 0, 0, 0, time.UTC),
				Type:  "MedicationStatement",
				End:   false,
				Value: new(models.MedicationStatement),
			},
		},
	}
	addSignificantBirthdayEvents(es, []int{40, 55, 65, 150})
	c.Assert(es.Events, HasLen, 5) // Note: Age 150 should not generate event since it is in future
	c.Assert(es.Events[0], DeepEquals, plugin.Event{
		Date:  time.Date(1985, time.January, 1, 12, 0, 0, 0, time.UTC),
		Type:  "Condition",
		End:   false,
		Value: new(models.Condition),
	})
	c.Assert(es.Events[1], DeepEquals, plugin.Event{
		Date:  time.Date(1990, time.March, 1, 12, 0, 0, 0, time.UTC),
		Type:  "Age",
		End:   false,
		Value: 40,
	})
	c.Assert(es.Events[2], DeepEquals, plugin.Event{
		Date:  time.Date(2005, time.March, 1, 12, 0, 0, 0, time.UTC),
		Type:  "Age",
		End:   false,
		Value: 55,
	})
	c.Assert(es.Events[3], DeepEquals, plugin.Event{
		Date:  time.Date(2010, time.February, 1, 12, 0, 0, 0, time.UTC),
		Type:  "MedicationStatement",
		End:   false,
		Value: new(models.MedicationStatement),
	})
	c.Assert(es.Events[4], DeepEquals, plugin.Event{
		Date:  time.Date(2015, time.March, 1, 12, 0, 0, 0, time.UTC),
		Type:  "Age",
		End:   false,
		Value: 65,
	})
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
	qURL, err := s.Service.getRequiredDataQueryURL("12345", "http://example.org/fhir")
	util.CheckErr(err)
	c.Assert(strings.HasPrefix(qURL, "http://example.org/fhir/Patient?"), Equals, true)
	qURL2, _ := url.Parse(qURL)
	c.Assert(qURL2.Query(), HasLen, 2)
	c.Assert(qURL2.Query().Get("_id"), Equals, "12345")
	c.Assert(qURL2.Query()["_revinclude"], HasLen, 1)
	c.Assert(qURL2.Query().Get("_revinclude"), Equals, "Condition:patient")
}

func (s *ServiceSuite) TestGetRequiredDataQueryURLForSimple(c *C) {
	s.Service.RegisterPlugin(simple.NewSimplePlugin())
	qURL, err := s.Service.getRequiredDataQueryURL("12345", "http://example.org/fhir")
	util.CheckErr(err)
	c.Assert(strings.HasPrefix(qURL, "http://example.org/fhir/Patient?"), Equals, true)
	qURL2, _ := url.Parse(qURL)
	c.Assert(qURL2.Query(), HasLen, 2)
	c.Assert(qURL2.Query().Get("_id"), Equals, "12345")
	c.Assert(qURL2.Query()["_revinclude"], HasLen, 2)
	var cFound, mFound bool
	for _, v := range qURL2.Query()["_revinclude"] {
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
	qURL, err := s.Service.getRequiredDataQueryURL("12345", "http://example.org/fhir")
	util.CheckErr(err)
	c.Assert(strings.HasPrefix(qURL, "http://example.org/fhir/Patient?"), Equals, true)
	qURL2, _ := url.Parse(qURL)
	c.Assert(qURL2.Query()["_revinclude"], HasLen, 2)
	var cFound, mFound bool
	for _, v := range qURL2.Query()["_revinclude"] {
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
			Pie:   assessment.NewPie(s.Server.URL + "/Patient/12345"),
		}, {
			AsOf:  time.Date(2000, 7, 14, 16, 0, 0, 0, time.UTC),
			Score: &one,
			Pie:   assessment.NewPie(s.Server.URL + "/Patient/12345"),
		}, {
			AsOf:  time.Date(2012, 1, 1, 11, 0, 0, 0, time.UTC),
			Score: &three,
			Pie:   assessment.NewPie(s.Server.URL + "/Patient/12345"),
		}, {
			AsOf:  time.Date(2013, 1, 1, 11, 0, 0, 0, time.UTC),
			Score: &four,
			Pie:   assessment.NewPie(s.Server.URL + "/Patient/12345"),
		}, {
			AsOf:  time.Date(2014, 2, 3, 10, 0, 0, 0, time.UTC),
			Score: &two,
			Pie:   assessment.NewPie(s.Server.URL + "/Patient/12345"),
		},
	}

	pieBasisURL := "http://example.org/Pie"
	simplePlugin := simple.NewSimplePlugin()
	bundle := buildRiskAssessmentBundle("12345", results, pieBasisURL, simplePlugin)

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
		c.Assert(ra.Basis[0].Reference, Equals, pieBasisURL+"/"+result.Pie.Id.Hex())
		c.Assert(ra.Date.Time, DeepEquals, result.AsOf)
		c.Assert(ra.Date.Precision, Equals, models.Precision(models.Timestamp))
		c.Assert(*ra.Method, DeepEquals, simplePlugin.Config().Method)
		c.Assert(ra.Prediction, HasLen, 1)
		c.Assert(*ra.Prediction[0].Outcome, DeepEquals, simplePlugin.Config().PredictedOutcome)
		c.Assert(*ra.Prediction[0].ProbabilityDecimal, Equals, float64(*result.Score))
		c.Assert(ra.Subject.Reference, Equals, "Patient/12345")
	}
}
