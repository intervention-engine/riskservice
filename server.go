package main

import (
	"github.com/intervention-engine/riskservice/server"
	"github.com/labstack/echo"
	"gopkg.in/mgo.v2"
	"os"
)

func main() {
	// Check for a linked MongoDB container if we are running in Docker
	mongoHost := os.Getenv("MONGO_PORT_27017_TCP_ADDR")
	if mongoHost == "" {
		mongoHost = "localhost"
	}
	e := echo.New()
	session, err := mgo.Dial(mongoHost)
	if err != nil {
		panic("Can't connect to the database")
	}
	defer session.Close()
	server.RegisterRiskHandlers(e, session.DB("riskservice"), "http://localhost:9000/pies/")
	e.Run(":9000")
}
