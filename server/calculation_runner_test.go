package server

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/intervention-engine/fhir/models"
	"github.com/intervention-engine/riskservice/assessment"
	"github.com/pebbe/util"
	. "gopkg.in/check.v1"
	"gopkg.in/mgo.v2/dbtest"
)

type CalculationRunnerSuite struct {
	Server   *httptest.Server
	DBServer *dbtest.DBServer
}

func Test(t *testing.T) { TestingT(t) }

var _ = Suite(&CalculationRunnerSuite{})

func (crs *CalculationRunnerSuite) SetUpSuite(c *C) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.RequestURI, "RiskAssessment") {
			w.Header().Add("Location", "http://localhost/RiskAssessment/1/_history/1")
			fmt.Fprintln(w, "Created")
		}
	})
	crs.Server = httptest.NewServer(handler)

	crs.DBServer = &dbtest.DBServer{}
	crs.DBServer.SetPath(c.MkDir())
}

func (crs *CalculationRunnerSuite) TearDownTest(c *C) {
	crs.DBServer.Wipe()
}

func (crs *CalculationRunnerSuite) TearDownSuite(c *C) {
	crs.DBServer.Stop()
}

func MockRiskCalculation(fhirEndpointUrl, patientId string, ts time.Time) (*models.RiskAssessment, *assessment.Pie, error) {
	pie := assessment.NewPie("")
	pie.AddSlice("Humors", 50, 1)
	pie.AddSlice("Blood-letting", 50, 4)

	assessment := &models.RiskAssessment{}
	assessment.Date = &models.FHIRDateTime{Time: time.Now(), Precision: models.Timestamp}
	prediction := models.RiskAssessmentPredictionComponent{}
	strokeRisk := 10.0
	prediction.ProbabilityDecimal = &strokeRisk
	prediction.Outcome = &models.CodeableConcept{Text: "Stroke"}
	assessment.Prediction = []models.RiskAssessmentPredictionComponent{prediction}
	return assessment, pie, nil
}

func (crs *CalculationRunnerSuite) TestRunner(c *C) {
	requests := make(chan CalculationRequest)
	done := make(chan struct{})
	riskAssessments := []RiskAssessmentCalculation{MockRiskCalculation}
	session := crs.DBServer.Session()
	defer session.Close()
	db := session.DB("test")
	var wg sync.WaitGroup
	wg.Add(1)
	go Runner(requests, done, "http://pie.org", riskAssessments, db, &wg)
	requests <- CalculationRequest{"http://fhir.org", "foo", time.Now(), time.Now()}
	requests <- CalculationRequest{"http://fhir.org", "bar", time.Now(), time.Now()}
	time.Sleep(1 * time.Second)
	requests <- CalculationRequest{"http://fhir.org", "foo", time.Now(), time.Now()}
	time.Sleep(2500 * time.Millisecond)
	count, _ := db.C("pies").Count()
	// Should have the pie only for bar
	c.Assert(count, Equals, 1)
	time.Sleep(1 * time.Second)
	count, _ = db.C("pies").Count()
	// Should have all pies
	c.Assert(count, Equals, 3)
	close(done)
	wg.Wait()
	// It actually stopped the runner
	c.Succeed()
}

func (crs *CalculationRunnerSuite) TestCreateRiskAssessment(c *C) {
	session := crs.DBServer.Session()
	defer session.Close()
	db := session.DB("test")
	err := CreateRiskAssessment(crs.Server.URL, "foo", "http://pie.org", MockRiskCalculation, db, time.Date(2015, time.August, 1, 0, 0, 0, 0, time.UTC))
	util.CheckErr(err)
	count, _ := db.C("pies").Count()
	c.Assert(count, Equals, 1)
	pie := &assessment.Pie{}
	db.C("pies").Find(nil).One(pie)
	c.Assert(len(pie.Slices), Equals, 2)
}
