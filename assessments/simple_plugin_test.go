package assessments

import (
	"time"

	"github.com/intervention-engine/fhir/models"
	"github.com/intervention-engine/riskservice/plugin"
	. "gopkg.in/check.v1"
)

type SimplePluginSuite struct {
	Plugin          *SimplePlugin
	FHIREndpointURL string
}

var _ = Suite(&SimplePluginSuite{})

func (cs *SimplePluginSuite) SetUpSuite(c *C) {
	cs.Plugin = &SimplePlugin{}
	cs.FHIREndpointURL = "http://example.org/fhir"
}

func (cs *SimplePluginSuite) TearDownSuite(c *C) {
	cs.Plugin = nil
}

func (cs *SimplePluginSuite) TestPatientWithNoConditionsAndNoMeds(c *C) {
	birthDate := &models.FHIRDateTime{Time: time.Date(1940, time.July, 1, 0, 0, 0, 0, time.UTC), Precision: models.Date}
	patient := &models.Patient{Gender: "female", BirthDate: birthDate}
	patient.Id = "1223"
	es := plugin.NewEventStream(patient)
	results, err := cs.Plugin.Calculate(es, cs.FHIREndpointURL)
	c.Assert(err, IsNil)
	// We expect one result with an asof time of now(ish)
	c.Assert(results, HasLen, 1)
	// Since we don't have a good way of knowing the exact expected timestamp, just check that it is recent and then
	// take it as an expected value so it doesn't fail the assertResult call
	t := results[0].AsOf
	c.Assert(time.Since(t).Minutes() < 1.0, Equals, true)
	cs.assertResult(c, results[0], t, 0, "1223", 0, 0)
}

func (cs *SimplePluginSuite) TestPatientWithSomeConditionsAndNoMeds(c *C) {
	birthDate := &models.FHIRDateTime{Time: time.Date(1940, time.July, 1, 0, 0, 0, 0, time.UTC), Precision: models.Date}
	patient := &models.Patient{Gender: "female", BirthDate: birthDate}
	patient.Id = "1223"
	es := plugin.NewEventStream(patient)
	es.Events = append(es.Events, conditionEvent("1", "Atrial Fibrillation", "427.31", time.Date(2010, time.February, 15, 15, 0, 0, 0, time.UTC)))
	es.Events = append(es.Events, conditionEvent("2", "Hypertension", "401.0", time.Date(2015, time.April, 15, 15, 0, 0, 0, time.UTC)))
	results, err := cs.Plugin.Calculate(es, cs.FHIREndpointURL)
	c.Assert(err, IsNil)
	c.Assert(results, HasLen, 2)
	cs.assertResult(c, results[0], time.Date(2010, time.February, 15, 15, 0, 0, 0, time.UTC), 1, "1223", 1, 0)
	cs.assertResult(c, results[1], time.Date(2015, time.April, 15, 15, 0, 0, 0, time.UTC), 2, "1223", 2, 0)
}

func (cs *SimplePluginSuite) TestPatientWithNoConditionsAndSomeMeds(c *C) {
	birthDate := &models.FHIRDateTime{Time: time.Date(1940, time.July, 1, 0, 0, 0, 0, time.UTC), Precision: models.Date}
	patient := &models.Patient{Gender: "female", BirthDate: birthDate}
	patient.Id = "1223"
	es := plugin.NewEventStream(patient)
	es.Events = append(es.Events, medicationEvent("1", "Aspirin", "1191", time.Date(2010, time.April, 15, 15, 30, 0, 0, time.UTC)))
	es.Events = append(es.Events, medicationEvent("2", "Lisinopril", "104377", time.Date(2015, time.April, 15, 15, 30, 0, 0, time.UTC)))
	results, err := cs.Plugin.Calculate(es, cs.FHIREndpointURL)
	c.Assert(err, IsNil)
	c.Assert(results, HasLen, 2)
	cs.assertResult(c, results[0], time.Date(2010, time.April, 15, 15, 30, 0, 0, time.UTC), 1, "1223", 0, 1)
	cs.assertResult(c, results[1], time.Date(2015, time.April, 15, 15, 30, 0, 0, time.UTC), 2, "1223", 0, 2)
}

func (cs *SimplePluginSuite) TestPatientWithSomeConditionsAndSomeMeds(c *C) {
	birthDate := &models.FHIRDateTime{Time: time.Date(1940, time.July, 1, 0, 0, 0, 0, time.UTC), Precision: models.Date}
	patient := &models.Patient{Gender: "female", BirthDate: birthDate}
	patient.Id = "1223"
	es := plugin.NewEventStream(patient)
	es.Events = append(es.Events, conditionEvent("1", "Atrial Fibrillation", "427.31", time.Date(2010, time.February, 15, 15, 0, 0, 0, time.UTC)))
	es.Events = append(es.Events, medicationEvent("2", "Aspirin", "1191", time.Date(2010, time.February, 15, 15, 30, 0, 0, time.UTC)))
	es.Events = append(es.Events, conditionEvent("3", "Hypertension", "401.0", time.Date(2015, time.April, 15, 15, 0, 0, 0, time.UTC)))
	es.Events = append(es.Events, medicationEvent("4", "Lisinopril", "104377", time.Date(2015, time.April, 15, 15, 30, 0, 0, time.UTC)))
	results, err := cs.Plugin.Calculate(es, cs.FHIREndpointURL)
	c.Assert(err, IsNil)
	c.Assert(results, HasLen, 4)
	cs.assertResult(c, results[0], time.Date(2010, time.February, 15, 15, 0, 0, 0, time.UTC), 1, "1223", 1, 0)
	cs.assertResult(c, results[1], time.Date(2010, time.February, 15, 15, 30, 0, 0, time.UTC), 2, "1223", 1, 1)
	cs.assertResult(c, results[2], time.Date(2015, time.April, 15, 15, 0, 0, 0, time.UTC), 3, "1223", 2, 1)
	cs.assertResult(c, results[3], time.Date(2015, time.April, 15, 15, 30, 0, 0, time.UTC), 4, "1223", 2, 2)
}

func (cs *SimplePluginSuite) TestPatientWithEndingConditionsAndEndingMeds(c *C) {
	birthDate := &models.FHIRDateTime{Time: time.Date(1940, time.July, 1, 0, 0, 0, 0, time.UTC), Precision: models.Date}
	patient := &models.Patient{Gender: "female", BirthDate: birthDate}
	patient.Id = "1223"
	es := plugin.NewEventStream(patient)
	afibStart, afibEnd := conditionStartAndEndEvents("1", "Atrial Fibrillation", "427.31", time.Date(2010, time.February, 15, 15, 0, 0, 0, time.UTC), time.Date(2015, time.May, 15, 15, 0, 0, 0, time.UTC))
	aspirinStart, aspirinEnd := medicationStartAndEndEvents("2", "Aspirin", "1191", time.Date(2010, time.February, 15, 15, 30, 0, 0, time.UTC), time.Date(2015, time.May, 15, 15, 30, 0, 0, time.UTC))
	es.Events = append(es.Events, afibStart)
	es.Events = append(es.Events, aspirinStart)
	es.Events = append(es.Events, conditionEvent("3", "Hypertension", "401.0", time.Date(2015, time.April, 15, 15, 0, 0, 0, time.UTC)))
	es.Events = append(es.Events, medicationEvent("4", "Lisinopril", "104377", time.Date(2015, time.April, 15, 15, 30, 0, 0, time.UTC)))
	es.Events = append(es.Events, afibEnd)
	es.Events = append(es.Events, aspirinEnd)
	results, err := cs.Plugin.Calculate(es, cs.FHIREndpointURL)
	c.Assert(err, IsNil)
	c.Assert(results, HasLen, 6)
	cs.assertResult(c, results[0], time.Date(2010, time.February, 15, 15, 0, 0, 0, time.UTC), 1, "1223", 1, 0)
	cs.assertResult(c, results[1], time.Date(2010, time.February, 15, 15, 30, 0, 0, time.UTC), 2, "1223", 1, 1)
	cs.assertResult(c, results[2], time.Date(2015, time.April, 15, 15, 0, 0, 0, time.UTC), 3, "1223", 2, 1)
	cs.assertResult(c, results[3], time.Date(2015, time.April, 15, 15, 30, 0, 0, time.UTC), 4, "1223", 2, 2)
	cs.assertResult(c, results[4], time.Date(2015, time.May, 15, 15, 0, 0, 0, time.UTC), 3, "1223", 1, 2)
	cs.assertResult(c, results[5], time.Date(2015, time.May, 15, 15, 30, 0, 0, time.UTC), 2, "1223", 1, 1)
}

func (cs *SimplePluginSuite) TestPatientWithDuplicates(c *C) {
	birthDate := &models.FHIRDateTime{Time: time.Date(1940, time.July, 1, 0, 0, 0, 0, time.UTC), Precision: models.Date}
	patient := &models.Patient{Gender: "female", BirthDate: birthDate}
	patient.Id = "1223"
	es := plugin.NewEventStream(patient)
	es.Events = append(es.Events, conditionEvent("1", "Atrial Fibrillation", "427.31", time.Date(2010, time.February, 15, 15, 0, 0, 0, time.UTC)))
	es.Events = append(es.Events, medicationEvent("2", "Aspirin", "1191", time.Date(2010, time.February, 15, 15, 30, 0, 0, time.UTC)))
	es.Events = append(es.Events, conditionEvent("3", "Hypertension", "401.0", time.Date(2015, time.April, 15, 15, 0, 0, 0, time.UTC)))
	es.Events = append(es.Events, medicationEvent("4", "Lisinopril", "104377", time.Date(2015, time.April, 15, 15, 30, 0, 0, time.UTC)))
	es.Events = append(es.Events, conditionEvent("5", "Hypertension", "401.0", time.Date(2015, time.May, 1, 15, 0, 0, 0, time.UTC)))
	es.Events = append(es.Events, medicationEvent("6", "Lisinopril", "104377", time.Date(2015, time.May, 1, 15, 30, 0, 0, time.UTC)))

	results, err := cs.Plugin.Calculate(es, cs.FHIREndpointURL)
	c.Assert(err, IsNil)
	c.Assert(results, HasLen, 6)
	cs.assertResult(c, results[0], time.Date(2010, time.February, 15, 15, 0, 0, 0, time.UTC), 1, "1223", 1, 0)
	cs.assertResult(c, results[1], time.Date(2010, time.February, 15, 15, 30, 0, 0, time.UTC), 2, "1223", 1, 1)
	cs.assertResult(c, results[2], time.Date(2015, time.April, 15, 15, 0, 0, 0, time.UTC), 3, "1223", 2, 1)
	cs.assertResult(c, results[3], time.Date(2015, time.April, 15, 15, 30, 0, 0, time.UTC), 4, "1223", 2, 2)
	cs.assertResult(c, results[4], time.Date(2015, time.May, 1, 15, 0, 0, 0, time.UTC), 4, "1223", 2, 2)
	cs.assertResult(c, results[5], time.Date(2015, time.May, 1, 15, 30, 0, 0, time.UTC), 4, "1223", 2, 2)
}

func (cs *SimplePluginSuite) TestFutureEventsAreIgnored(c *C) {
	birthDate := &models.FHIRDateTime{Time: time.Date(1940, time.July, 1, 0, 0, 0, 0, time.UTC), Precision: models.Date}
	patient := &models.Patient{Gender: "female", BirthDate: birthDate}
	patient.Id = "1223"
	es := plugin.NewEventStream(patient)
	afibStart, afibEnd := conditionStartAndEndEvents("1", "Atrial Fibrillation", "427.31", time.Date(2010, time.February, 15, 15, 0, 0, 0, time.UTC), time.Date(2035, time.May, 15, 15, 0, 0, 0, time.UTC))
	aspirinStart, aspirinEnd := medicationStartAndEndEvents("2", "Aspirin", "1191", time.Date(2010, time.February, 15, 15, 30, 0, 0, time.UTC), time.Date(2015, time.May, 15, 15, 30, 0, 0, time.UTC))
	es.Events = append(es.Events, afibStart)
	es.Events = append(es.Events, aspirinStart)
	es.Events = append(es.Events, conditionEvent("3", "Hypertension", "401.0", time.Date(2015, time.April, 15, 15, 0, 0, 0, time.UTC)))
	es.Events = append(es.Events, medicationEvent("4", "Lisinopril", "104377", time.Date(2015, time.April, 15, 15, 30, 0, 0, time.UTC)))
	es.Events = append(es.Events, aspirinEnd)
	// This future event should not be counted!
	es.Events = append(es.Events, conditionEvent("5", "Stroke", "434.91", time.Date(2030, time.June, 15, 15, 0, 0, 0, time.UTC)))
	// This future event end should not be counted!
	es.Events = append(es.Events, afibEnd)
	results, err := cs.Plugin.Calculate(es, cs.FHIREndpointURL)
	c.Assert(err, IsNil)
	c.Assert(results, HasLen, 5)
	cs.assertResult(c, results[0], time.Date(2010, time.February, 15, 15, 0, 0, 0, time.UTC), 1, "1223", 1, 0)
	cs.assertResult(c, results[1], time.Date(2010, time.February, 15, 15, 30, 0, 0, time.UTC), 2, "1223", 1, 1)
	cs.assertResult(c, results[2], time.Date(2015, time.April, 15, 15, 0, 0, 0, time.UTC), 3, "1223", 2, 1)
	cs.assertResult(c, results[3], time.Date(2015, time.April, 15, 15, 30, 0, 0, time.UTC), 4, "1223", 2, 2)
	cs.assertResult(c, results[4], time.Date(2015, time.May, 15, 15, 30, 0, 0, time.UTC), 3, "1223", 2, 1)
}

func (cs *SimplePluginSuite) TestNonSignificantEvents(c *C) {
	birthDate := &models.FHIRDateTime{Time: time.Date(1940, time.July, 1, 0, 0, 0, 0, time.UTC), Precision: models.Date}
	patient := &models.Patient{Gender: "female", BirthDate: birthDate}
	patient.Id = "1223"
	es := plugin.NewEventStream(patient)
	es.Events = append(es.Events, conditionEvent("1", "Atrial Fibrillation", "427.31", time.Date(2010, time.February, 15, 15, 0, 0, 0, time.UTC)))
	es.Events = append(es.Events, medicationEvent("2", "Aspirin", "1191", time.Date(2010, time.February, 15, 15, 30, 0, 0, time.UTC)))
	es.Events = append(es.Events, encounterEvent("3", "Consultation", "11429006", time.Date(2012, time.March, 15, 15, 0, 0, 0, time.UTC)))
	weightFloat := float64(163)
	weight := models.Quantity{Value: &weightFloat, Unit: "lb_av"}
	es.Events = append(es.Events, observationEvent("4", "Body Weight", "29463-7", weight, time.Date(2012, time.March, 15, 15, 30, 0, 0, time.UTC)))
	es.Events = append(es.Events, conditionEvent("5", "Hypertension", "401.0", time.Date(2015, time.April, 15, 15, 0, 0, 0, time.UTC)))
	es.Events = append(es.Events, medicationEvent("6", "Lisinopril", "104377", time.Date(2015, time.April, 15, 15, 30, 0, 0, time.UTC)))
	results, err := cs.Plugin.Calculate(es, cs.FHIREndpointURL)
	c.Assert(err, IsNil)
	c.Assert(results, HasLen, 4)
	cs.assertResult(c, results[0], time.Date(2010, time.February, 15, 15, 0, 0, 0, time.UTC), 1, "1223", 1, 0)
	cs.assertResult(c, results[1], time.Date(2010, time.February, 15, 15, 30, 0, 0, time.UTC), 2, "1223", 1, 1)
	cs.assertResult(c, results[2], time.Date(2015, time.April, 15, 15, 0, 0, 0, time.UTC), 3, "1223", 2, 1)
	cs.assertResult(c, results[3], time.Date(2015, time.April, 15, 15, 30, 0, 0, time.UTC), 4, "1223", 2, 2)
}

func (cs *SimplePluginSuite) assertResult(c *C, result plugin.RiskServiceCalculationResult, asOf time.Time, score int, patientID string, conditions, medications int) {
	c.Assert(result.AsOf, DeepEquals, asOf)
	c.Assert(*result.Score, Equals, score)
	c.Assert(result.ProbabilityDecimal, IsNil)
	pie := result.Pie
	c.Assert(pie.Patient, Equals, cs.FHIREndpointURL+"/Patient/"+patientID)
	c.Assert(pie.Slices, HasLen, 2)
	c.Assert(pie.Slices[0].Name, Equals, "Conditions")
	c.Assert(pie.Slices[0].Weight, Equals, 50)
	c.Assert(pie.Slices[0].MaxValue, Equals, 5)
	c.Assert(pie.Slices[0].Value, Equals, conditions)
	c.Assert(pie.Slices[1].Name, Equals, "Medications")
	c.Assert(pie.Slices[1].Weight, Equals, 50)
	c.Assert(pie.Slices[1].MaxValue, Equals, 5)
	c.Assert(pie.Slices[1].Value, Equals, medications)
}
