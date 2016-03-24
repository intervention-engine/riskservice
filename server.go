package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"github.com/intervention-engine/fhir/models"
	"github.com/intervention-engine/fhir/upload"
	"github.com/intervention-engine/riskservice/assessments"
	"github.com/intervention-engine/riskservice/server"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"gopkg.in/mgo.v2"
)

func main() {
	// Check for a linked MongoDB container if we are running in Docker
	mongoHost := os.Getenv("MONGO_PORT_27017_TCP_ADDR")
	if mongoHost == "" {
		mongoHost = "localhost"
	}
	registerURL := flag.String("registerURL", "", "Register a FHIR Subscription to the specified URL")
	registerENV := flag.String("registerENV", "", "Register a FHIR Subscription to the the Docker environment variable IE_PORT_3001_TCP*")
	flag.Parse()
	parsedURL := *registerURL
	if parsedURL != "" {
		registerServer(parsedURL)
	}
	if registerENV != nil {
		registerServer(fmt.Sprintf("http://%s:%s", os.Getenv("IE_PORT_3001_TCP_ADDR"), os.Getenv("IE_PORT_3001_TCP_PORT")))
	}

	e := echo.New()
	session, err := mgo.Dial(mongoHost)
	if err != nil {
		panic("Can't connect to the database")
	}
	defer session.Close()

	basePieURL := discoverSelf() + "pies"
	db := session.DB("riskservice")
	service := server.NewReferenceRiskService(db)
	service.RegisterPlugin(assessments.NewCHA2DS2VAScPlugin())
	service.RegisterPlugin(assessments.NewSimplePlugin())
	fnDelayer := server.NewFunctionDelayer(3 * time.Second)
	server.RegisterRoutes(e, db, basePieURL, service, fnDelayer)
	e.Use(middleware.Logger())
	e.Run(":9000")
}

func discoverSelf() string {
	var ip net.IP
	var selfURL string
	host, err := os.Hostname()
	if err != nil {
		panic(err)
	}
	addrs, err := net.LookupIP(host)
	if err != nil {
		log.Println("Unable to lookup IP based on hostname, defaulting to localhost.")
		selfURL = "http://localhost:9000/"
	}
	for _, addr := range addrs {
		if ipv4 := addr.To4(); ipv4 != nil {
			ip = ipv4
			selfURL = "http://" + ip.String() + ":9000/"
		}
	}
	return selfURL
}

func registerServer(registerString string) {
	channel := &models.SubscriptionChannelComponent{Type: "rest-hook", Endpoint: discoverSelf() + "calculate"}
	sub := models.Subscription{Channel: channel}
	upload.UploadResource(&sub, registerString)
}
