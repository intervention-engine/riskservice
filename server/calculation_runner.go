package server

import (
	"sync"
	"time"

	"github.com/intervention-engine/fhir/models"
	"github.com/intervention-engine/fhir/upload"
	"github.com/intervention-engine/riskservice/assessment"
	"gopkg.in/mgo.v2"
)

// Interface that all risk assessments should conform to
type RiskAssessmentCalculation func(fhirEndpointUrl, patientId string, ts time.Time) (*models.RiskAssessment, *assessment.Pie, error)

// Holder for web requests that come in for risk calculations
type CalculationRequest struct {
	FHIREndpointURL string
	PatientID       string
	RiskAt          time.Time
	ArrivedAt       time.Time
}

// Watches the requestChan for calculation requests. Will buffer requests for
// patients until it hasn't recieved a calculation request for that patient for
// 3 seconds. Will then kick off perform all risk assessments for that patient.
// Should be run as a goroutine. Shutdown the goroutine by closing the done
// channel.
func Runner(requestChan <-chan CalculationRequest, done <-chan struct{}, basePieURL string, riskAssessments []RiskAssessmentCalculation, db *mgo.Database, wg *sync.WaitGroup) {
	requestMap := make(map[string][]CalculationRequest)
	var quit bool
	defer wg.Done()
	for {
		select {
		case cr := <-requestChan:
			requestList, ok := requestMap[cr.PatientID]
			if ok {
				requestMap[cr.PatientID] = append(requestList, cr)
			} else {
				requestMap[cr.PatientID] = []CalculationRequest{cr}
			}
		case <-done:
			// We are done so wait for 3 seconds so we can clear the entire requestMap
			time.Sleep(3 * time.Second)
			quit = true
		case <-time.After(500 * time.Millisecond):
			//Do nothing, this lets us check the requestMap every 500ms even if no
			//new values were passed in.
		}
		for patientID, crList := range requestMap {
			newestCr := crList[len(crList)-1]
			now := time.Now()
			if now.Sub(newestCr.ArrivedAt) > 3*time.Second {
				for _, c := range crList {
					for _, rac := range riskAssessments {
						CreateRiskAssessment(c.FHIREndpointURL, c.PatientID, basePieURL, rac, db, c.RiskAt)
					}
				}
				delete(requestMap, patientID)
			}
		}
		if quit {
			return
		}
	}
}

func CreateRiskAssessment(fhirEndpointUrl, patientId, basePieUrl string, rac RiskAssessmentCalculation, db *mgo.Database, ts time.Time) error {
	ra, pie, err := rac(fhirEndpointUrl, patientId, ts)
	if err != nil {
		return err
	}
	pieCollection := db.C("pies")
	err = pieCollection.Insert(pie)
	if err != nil {
		return err
	}
	ra.Basis = []models.Reference{models.Reference{Reference: basePieUrl + pie.Id.Hex()}}
	_, err = upload.UploadResource(ra, fhirEndpointUrl)
	if err != nil {
		return err
	}

	return nil
}
