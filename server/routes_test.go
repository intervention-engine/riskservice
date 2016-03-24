package server

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/intervention-engine/riskservice/plugin"
	"github.com/labstack/echo"
	"github.com/pebbe/util"
	. "gopkg.in/check.v1"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/dbtest"
)

type RoutesSuite struct {
	DBServer    *dbtest.DBServer
	Database    *mgo.Database
	Server      *httptest.Server
	MockService *MockService
}

func Test(t *testing.T) { TestingT(t) }

var _ = Suite(&RoutesSuite{})

func (r *RoutesSuite) SetUpSuite(c *C) {
	r.DBServer = &dbtest.DBServer{}
	r.DBServer.SetPath(c.MkDir())
}

func (r *RoutesSuite) SetUpTest(c *C) {
	r.Database = r.DBServer.Session().DB("test")
	r.MockService = &MockService{}
	e := echo.New()
	r.Server = httptest.NewServer(e)
	RegisterRoutes(e, r.Database, "http://foo.com", r.MockService, NewFunctionDelayer(500*time.Millisecond))
}

func (r *RoutesSuite) TearDownTest(c *C) {
	r.Server.Close()
	r.Database.Session.Close()
	r.DBServer.Wipe()
}

func (r *RoutesSuite) TearDownSuite(c *C) {
	r.DBServer.Stop()
}

func (r *RoutesSuite) TestPieRoute(c *C) {
	// Insert the pie in the database
	patientURL := "http://testurl.org"
	pie := plugin.NewPie(patientURL)
	r.Database.C("pies").Insert(pie)

	// Now get the pie from the server
	pieURL := fmt.Sprintf("%s/pies/%s", r.Server.URL, pie.Id.Hex())
	resp, err := http.Get(pieURL)
	util.CheckErr(err)
	c.Assert(resp.StatusCode, Equals, http.StatusOK)

	// Does the pie data contain the patient URL?
	defer resp.Body.Close()
	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	responseBody := buf.String()
	c.Assert(strings.Contains(responseBody, patientURL), Equals, true)
}

func (r *RoutesSuite) TestCalculateRoute(c *C) {
	patientID := "123"
	fhirEndpointURL := "http://example.org"

	// Now post the calculate request
	r.postCalculate(c, patientID, fhirEndpointURL)

	// It shouldn't calculate right away
	time.Sleep(100 * time.Millisecond)
	r.MockService.AssertCalls(c)

	// But after 600ms it should!
	time.Sleep(500 * time.Millisecond)
	r.MockService.AssertCalls(c,
		MockServiceCallParams{patientID: patientID, fhirEndpointURL: fhirEndpointURL, basePieURL: "http://foo.com"})
}

func (r *RoutesSuite) TestMultipleCalculateRequests(c *C) {
	// Now post the calculate request
	r.postCalculate(c, "123", "http://example.org/fhir")
	r.postCalculate(c, "123", "http://example.org/fhir")
	r.postCalculate(c, "456", "http://example.org/fhir2")
	time.Sleep(200 * time.Millisecond)
	r.postCalculate(c, "123", "http://example.org/fhir")

	// It shouldn't calculate anything right away
	time.Sleep(100 * time.Millisecond)
	r.MockService.AssertCalls(c)

	// But after 600ms it should have calculated patient 456
	time.Sleep(300 * time.Millisecond)
	r.MockService.AssertCalls(c,
		MockServiceCallParams{patientID: "456", fhirEndpointURL: "http://example.org/fhir2", basePieURL: "http://foo.com"})

	// And after 800ms it should have calculated patient 123 too
	time.Sleep(200 * time.Millisecond)
	r.MockService.AssertCalls(c,
		MockServiceCallParams{patientID: "456", fhirEndpointURL: "http://example.org/fhir2", basePieURL: "http://foo.com"},
		MockServiceCallParams{patientID: "123", fhirEndpointURL: "http://example.org/fhir", basePieURL: "http://foo.com"})

}

func (r *RoutesSuite) postCalculate(c *C, patientID, fhirEndpointURL string) {
	// Post the calculate request
	calcURL := fmt.Sprintf("%s/calculate", r.Server.URL)
	formData := url.Values{}
	formData.Set("patientId", patientID)
	formData.Set("fhirEndpointUrl", fhirEndpointURL)
	resp, err := http.PostForm(calcURL, formData)
	util.CheckErr(err)
	c.Assert(resp.StatusCode, Equals, http.StatusOK)
}

type MockService struct {
	sync.Mutex
	Calls []MockServiceCallParams
}

// Calculate makes MockService fulfill the RiskService interface
func (m *MockService) Calculate(patientID string, fhirEndpointURL string, basePieURL string) error {
	params := MockServiceCallParams{patientID: patientID, fhirEndpointURL: fhirEndpointURL, basePieURL: basePieURL}
	m.Lock()
	defer m.Unlock()
	m.Calls = append(m.Calls, params)
	return nil
}

func (m *MockService) reset() {
	m.Lock()
	defer m.Unlock()
	m.Calls = nil
}

func (m *MockService) AssertCalls(c *C, calls ...MockServiceCallParams) {
	m.Lock()
	defer m.Unlock()

	c.Assert(m.Calls, HasLen, len(calls))
	for i := range m.Calls {
		c.Assert(m.Calls[i].patientID, Equals, calls[i].patientID)
		c.Assert(m.Calls[i].fhirEndpointURL, Equals, calls[i].fhirEndpointURL)
		c.Assert(m.Calls[i].basePieURL, Equals, calls[i].basePieURL)
	}
}

type MockServiceCallParams struct {
	patientID       string
	fhirEndpointURL string
	basePieURL      string
}
