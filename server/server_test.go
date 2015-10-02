package server

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/intervention-engine/riskservice/assessment"
	"github.com/labstack/echo"
	"github.com/pebbe/util"
	. "gopkg.in/check.v1"
	"gopkg.in/mgo.v2/dbtest"
)

type ServerSuite struct {
	DBServer *dbtest.DBServer
}

var _ = Suite(&ServerSuite{})

func (s *ServerSuite) SetUpSuite(c *C) {
	s.DBServer = &dbtest.DBServer{}
	s.DBServer.SetPath(c.MkDir())
}

func (s *ServerSuite) TearDownTest(c *C) {
	s.DBServer.Wipe()
}

func (s *ServerSuite) TearDownSuite(c *C) {
	s.DBServer.Stop()
}

func (s *ServerSuite) TestRegisterRiskHandlers(c *C) {
	patientUrl := "http://testurl.org"
	e := echo.New()
	session := s.DBServer.Session()
	defer session.Close()
	db := session.DB("test")
	RegisterRiskHandlers(e, db, "http://foo.com", make(chan CalculationRequest))
	server := httptest.NewServer(e)
	pie := assessment.NewPie(patientUrl)
	db.C("pies").Insert(pie)
	pieUrl := fmt.Sprintf("%s/pies/%s", server.URL, pie.Id.Hex())
	fmt.Println(pieUrl)
	resp, err := http.Get(pieUrl)
	util.CheckErr(err)
	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	responseBody := buf.String()
	fmt.Println(responseBody)
	c.Assert(strings.Contains(responseBody, patientUrl), Equals, true)
}
