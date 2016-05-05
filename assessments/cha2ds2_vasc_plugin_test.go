package assessments

import (
	"testing"
	"time"

	"github.com/intervention-engine/fhir/models"
	"github.com/intervention-engine/riskservice/plugin"
	. "gopkg.in/check.v1"
)

type CHA2DS2VAScPluginSuite struct {
	Plugin          *CHA2DS2VAScPlugin
	FHIREndpointURL string
}

func Test(t *testing.T) { TestingT(t) }

var _ = Suite(&CHA2DS2VAScPluginSuite{})

func (cs *CHA2DS2VAScPluginSuite) SetUpSuite(c *C) {
	cs.Plugin = &CHA2DS2VAScPlugin{}
	cs.FHIREndpointURL = "http://example.org/fhir"
}

func (cs *CHA2DS2VAScPluginSuite) TearDownSuite(c *C) {
	cs.Plugin = nil
}

func (cs *CHA2DS2VAScPluginSuite) TestFemaleWithAFibOnly(c *C) {
	birthDate := &models.FHIRDateTime{Time: time.Date(1940, time.July, 1, 0, 0, 0, 0, time.UTC), Precision: models.Date}
	patient := &models.Patient{Gender: "female", BirthDate: birthDate}
	patient.Id = "1223"
	es := plugin.NewEventStream(patient)
	es.Events = append(es.Events, conditionEvent("1", "Atrial Fibrillation", "427.31", time.Date(1990, time.February, 15, 15, 0, 0, 0, time.UTC)))
	results, err := cs.Plugin.Calculate(es, cs.FHIREndpointURL)
	c.Assert(err, IsNil)
	c.Assert(results, HasLen, 1)
	cs.assertResult(c, results[0], time.Date(1990, time.February, 15, 15, 0, 0, 0, time.UTC), 1, 1.3, "1223", 0, 0, 0, 0, 0, 0, 1)
}

func (cs *CHA2DS2VAScPluginSuite) TestMaleWithAFibOnly(c *C) {
	birthDate := &models.FHIRDateTime{Time: time.Date(1980, time.July, 1, 0, 0, 0, 0, time.UTC), Precision: models.Date}
	patient := &models.Patient{Gender: "male", BirthDate: birthDate}
	patient.Id = "1223"
	es := plugin.NewEventStream(patient)
	es.Events = append(es.Events, conditionEvent("1", "Atrial Fibrillation", "427.31", time.Date(1990, time.February, 15, 15, 0, 0, 0, time.UTC)))
	results, err := cs.Plugin.Calculate(es, cs.FHIREndpointURL)
	c.Assert(err, IsNil)
	c.Assert(results, HasLen, 1)
	cs.assertResult(c, results[0], time.Date(1990, time.February, 15, 15, 0, 0, 0, time.UTC), 0, 0.0, "1223", 0, 0, 0, 0, 0, 0, 0)
}

func (cs *CHA2DS2VAScPluginSuite) TestFemaleWithEveryFactor(c *C) {
	birthDate := &models.FHIRDateTime{Time: time.Date(1940, time.July, 1, 0, 0, 0, 0, time.UTC), Precision: models.Date}
	patient := &models.Patient{Gender: "female", BirthDate: birthDate}
	patient.Id = "1223"
	es := plugin.NewEventStream(patient)
	es.Events = append(es.Events, conditionEvent("1", "Atrial Fibrillation", "427.31", time.Date(1990, time.February, 15, 15, 0, 0, 0, time.UTC)))
	es.Events = append(es.Events, conditionEvent("2", "Congestive Heart Failure", "428.0", time.Date(1993, time.March, 15, 15, 0, 0, 0, time.UTC)))
	es.Events = append(es.Events, conditionEvent("3", "Hypertension", "401.0", time.Date(1997, time.April, 15, 15, 0, 0, 0, time.UTC)))
	es.Events = append(es.Events, conditionEvent("4", "Diabetes", "250.0", time.Date(2000, time.May, 15, 15, 0, 0, 0, time.UTC)))
	es.Events = append(es.Events, conditionEvent("5", "Stroke", "434.91", time.Date(2004, time.June, 15, 15, 0, 0, 0, time.UTC)))
	es.Events = append(es.Events, ageEvent("6", 65, time.Date(2005, time.July, 1, 0, 0, 0, 0, time.UTC)))
	es.Events = append(es.Events, conditionEvent("7", "Vascular Disease", "443.9", time.Date(2007, time.July, 15, 15, 0, 0, 0, time.UTC)))
	es.Events = append(es.Events, ageEvent("8", 75, time.Date(2015, time.July, 1, 0, 0, 0, 0, time.UTC)))
	results, err := cs.Plugin.Calculate(es, cs.FHIREndpointURL)
	c.Assert(err, IsNil)
	c.Assert(results, HasLen, 8)
	cs.assertResult(c, results[0], time.Date(1990, time.February, 15, 15, 0, 0, 0, time.UTC), 1, 1.3, "1223", 0, 0, 0, 0, 0, 0, 1)
	cs.assertResult(c, results[1], time.Date(1993, time.March, 15, 15, 0, 0, 0, time.UTC), 2, 2.2, "1223", 1, 0, 0, 0, 0, 0, 1)
	cs.assertResult(c, results[2], time.Date(1997, time.April, 15, 15, 0, 0, 0, time.UTC), 3, 3.2, "1223", 1, 1, 0, 0, 0, 0, 1)
	cs.assertResult(c, results[3], time.Date(2000, time.May, 15, 15, 0, 0, 0, time.UTC), 4, 4.0, "1223", 1, 1, 1, 0, 0, 0, 1)
	cs.assertResult(c, results[4], time.Date(2004, time.June, 15, 15, 0, 0, 0, time.UTC), 6, 9.8, "1223", 1, 1, 1, 2, 0, 0, 1)
	cs.assertResult(c, results[5], time.Date(2005, time.July, 1, 0, 0, 0, 0, time.UTC), 7, 9.6, "1223", 1, 1, 1, 2, 0, 1, 1)
	cs.assertResult(c, results[6], time.Date(2007, time.July, 15, 15, 0, 0, 0, time.UTC), 8, 6.7, "1223", 1, 1, 1, 2, 1, 1, 1)
	cs.assertResult(c, results[7], time.Date(2015, time.July, 1, 0, 0, 0, 0, time.UTC), 9, 15.2, "1223", 1, 1, 1, 2, 1, 2, 1)
}

func (cs *CHA2DS2VAScPluginSuite) TestNonSignificantEvents(c *C) {
	birthDate := &models.FHIRDateTime{Time: time.Date(1940, time.July, 1, 0, 0, 0, 0, time.UTC), Precision: models.Date}
	patient := &models.Patient{Gender: "female", BirthDate: birthDate}
	patient.Id = "1223"
	es := plugin.NewEventStream(patient)
	es.Events = append(es.Events, conditionEvent("1", "Skin Rash", "782.1", time.Date(1985, time.January, 15, 15, 0, 0, 0, time.UTC)))
	es.Events = append(es.Events, conditionEvent("2", "Atrial Fibrillation", "427.31", time.Date(1990, time.February, 15, 15, 0, 0, 0, time.UTC)))
	weightFloat := float64(163)
	weight := models.Quantity{Value: &weightFloat, Unit: "lb_av"}
	es.Events = append(es.Events, observationEvent("3", "Body Weight", "29463-7", weight, time.Date(1991, time.February, 15, 15, 0, 0, 0, time.UTC)))
	es.Events = append(es.Events, conditionEvent("4", "Congestive Heart Failure", "428.0", time.Date(1993, time.March, 15, 15, 0, 0, 0, time.UTC)))
	es.Events = append(es.Events, conditionEvent("5", "Hypertension", "401.0", time.Date(1997, time.April, 15, 15, 0, 0, 0, time.UTC)))
	es.Events = append(es.Events, conditionEvent("6", "Diabetes", "250.0", time.Date(2000, time.May, 15, 15, 0, 0, 0, time.UTC)))
	es.Events = append(es.Events, conditionEvent("7", "Stroke", "434.91", time.Date(2004, time.June, 15, 15, 0, 0, 0, time.UTC)))
	es.Events = append(es.Events, ageEvent("8", 65, time.Date(2005, time.July, 1, 0, 0, 0, 0, time.UTC)))
	es.Events = append(es.Events, conditionEvent("9", "Ganglion Cyst", "727.4", time.Date(2006, time.July, 15, 15, 0, 0, 0, time.UTC)))
	es.Events = append(es.Events, conditionEvent("10", "Vascular Disease", "443.9", time.Date(2007, time.July, 15, 15, 0, 0, 0, time.UTC)))
	es.Events = append(es.Events, ageEvent("11", 75, time.Date(2015, time.July, 1, 0, 0, 0, 0, time.UTC)))
	es.Events = append(es.Events, encounterEvent("12", "Consultation", "11429006", time.Date(2016, time.February, 15, 15, 0, 0, 0, time.UTC)))
	weightFloat = float64(147)
	weight = models.Quantity{Value: &weightFloat, Unit: "lb_av"}
	es.Events = append(es.Events, observationEvent("13", "Body Weight", "29463-7", weight, time.Date(2016, time.February, 15, 15, 5, 0, 0, time.UTC)))
	results, err := cs.Plugin.Calculate(es, cs.FHIREndpointURL)
	c.Assert(err, IsNil)
	c.Assert(results, HasLen, 8)
	cs.assertResult(c, results[0], time.Date(1990, time.February, 15, 15, 0, 0, 0, time.UTC), 1, 1.3, "1223", 0, 0, 0, 0, 0, 0, 1)
	cs.assertResult(c, results[1], time.Date(1993, time.March, 15, 15, 0, 0, 0, time.UTC), 2, 2.2, "1223", 1, 0, 0, 0, 0, 0, 1)
	cs.assertResult(c, results[2], time.Date(1997, time.April, 15, 15, 0, 0, 0, time.UTC), 3, 3.2, "1223", 1, 1, 0, 0, 0, 0, 1)
	cs.assertResult(c, results[3], time.Date(2000, time.May, 15, 15, 0, 0, 0, time.UTC), 4, 4.0, "1223", 1, 1, 1, 0, 0, 0, 1)
	cs.assertResult(c, results[4], time.Date(2004, time.June, 15, 15, 0, 0, 0, time.UTC), 6, 9.8, "1223", 1, 1, 1, 2, 0, 0, 1)
	cs.assertResult(c, results[5], time.Date(2005, time.July, 1, 0, 0, 0, 0, time.UTC), 7, 9.6, "1223", 1, 1, 1, 2, 0, 1, 1)
	cs.assertResult(c, results[6], time.Date(2007, time.July, 15, 15, 0, 0, 0, time.UTC), 8, 6.7, "1223", 1, 1, 1, 2, 1, 1, 1)
	cs.assertResult(c, results[7], time.Date(2015, time.July, 1, 0, 0, 0, 0, time.UTC), 9, 15.2, "1223", 1, 1, 1, 2, 1, 2, 1)
}

func (cs *CHA2DS2VAScPluginSuite) TestFutureEventsAreIgnored(c *C) {
	birthDate := &models.FHIRDateTime{Time: time.Date(1940, time.July, 1, 0, 0, 0, 0, time.UTC), Precision: models.Date}
	patient := &models.Patient{Gender: "female", BirthDate: birthDate}
	patient.Id = "1223"
	es := plugin.NewEventStream(patient)
	es.Events = append(es.Events, conditionEvent("1", "Atrial Fibrillation", "427.31", time.Date(1990, time.February, 15, 15, 0, 0, 0, time.UTC)))
	es.Events = append(es.Events, conditionEvent("2", "Hypertension", "401.0", time.Date(1997, time.April, 15, 15, 0, 0, 0, time.UTC)))
	es.Events = append(es.Events, ageEvent("3", 65, time.Date(2005, time.July, 1, 0, 0, 0, 0, time.UTC)))
	es.Events = append(es.Events, conditionEvent("4", "Vascular Disease", "443.9", time.Date(2007, time.July, 15, 15, 0, 0, 0, time.UTC)))
	es.Events = append(es.Events, ageEvent("5", 75, time.Date(2015, time.July, 1, 0, 0, 0, 0, time.UTC)))
	// This future event should not be counted!
	es.Events = append(es.Events, conditionEvent("6", "Stroke", "434.91", time.Date(2035, time.June, 15, 15, 0, 0, 0, time.UTC)))
	results, err := cs.Plugin.Calculate(es, cs.FHIREndpointURL)
	c.Assert(err, IsNil)
	c.Assert(results, HasLen, 5)
	cs.assertResult(c, results[0], time.Date(1990, time.February, 15, 15, 0, 0, 0, time.UTC), 1, 1.3, "1223", 0, 0, 0, 0, 0, 0, 1)
	cs.assertResult(c, results[1], time.Date(1997, time.April, 15, 15, 0, 0, 0, time.UTC), 2, 2.2, "1223", 0, 1, 0, 0, 0, 0, 1)
	cs.assertResult(c, results[2], time.Date(2005, time.July, 1, 0, 0, 0, 0, time.UTC), 3, 3.2, "1223", 0, 1, 0, 0, 0, 1, 1)
	cs.assertResult(c, results[3], time.Date(2007, time.July, 15, 15, 0, 0, 0, time.UTC), 4, 4.0, "1223", 0, 1, 0, 0, 1, 1, 1)
	cs.assertResult(c, results[4], time.Date(2015, time.July, 1, 0, 0, 0, 0, time.UTC), 5, 6.7, "1223", 0, 1, 0, 0, 1, 2, 1)
}

func (cs *CHA2DS2VAScPluginSuite) TestFactorsBeforeAFib(c *C) {
	birthDate := &models.FHIRDateTime{Time: time.Date(1940, time.July, 1, 0, 0, 0, 0, time.UTC), Precision: models.Date}
	patient := &models.Patient{Gender: "female", BirthDate: birthDate}
	patient.Id = "1223"
	es := plugin.NewEventStream(patient)
	// These first two events shouldn't trigger risk calculation results since they are *before* afib
	es.Events = append(es.Events, ageEvent("1", 65, time.Date(2005, time.July, 1, 0, 0, 0, 0, time.UTC)))
	es.Events = append(es.Events, conditionEvent("2", "Congestive Heart Failure", "428.0", time.Date(2006, time.March, 15, 15, 0, 0, 0, time.UTC)))
	// Once afib is diagnosed, the previous events should already be factored in the score
	es.Events = append(es.Events, conditionEvent("3", "Atrial Fibrillation", "427.31", time.Date(2010, time.February, 15, 15, 0, 0, 0, time.UTC)))
	es.Events = append(es.Events, conditionEvent("4", "Diabetes", "250.0", time.Date(2012, time.May, 15, 15, 0, 0, 0, time.UTC)))
	es.Events = append(es.Events, ageEvent("5", 75, time.Date(2015, time.July, 1, 0, 0, 0, 0, time.UTC)))
	results, err := cs.Plugin.Calculate(es, cs.FHIREndpointURL)
	c.Assert(err, IsNil)
	c.Assert(results, HasLen, 3)
	cs.assertResult(c, results[0], time.Date(2010, time.February, 15, 15, 0, 0, 0, time.UTC), 3, 3.2, "1223", 1, 0, 0, 0, 0, 1, 1)
	cs.assertResult(c, results[1], time.Date(2012, time.May, 15, 15, 0, 0, 0, time.UTC), 4, 4.0, "1223", 1, 0, 1, 0, 0, 1, 1)
	cs.assertResult(c, results[2], time.Date(2015, time.July, 1, 0, 0, 0, 0, time.UTC), 5, 6.7, "1223", 1, 0, 1, 0, 0, 2, 1)
}

func (cs *CHA2DS2VAScPluginSuite) TestNoAFib(c *C) {
	birthDate := &models.FHIRDateTime{Time: time.Date(1940, time.July, 1, 0, 0, 0, 0, time.UTC), Precision: models.Date}
	patient := &models.Patient{Gender: "female", BirthDate: birthDate}
	patient.Id = "1223"
	es := plugin.NewEventStream(patient)
	// None of these events should matter because this score is only valid for patients with atrial fibrillation
	es.Events = append(es.Events, ageEvent("1", 65, time.Date(2005, time.July, 1, 0, 0, 0, 0, time.UTC)))
	es.Events = append(es.Events, conditionEvent("2", "Congestive Heart Failure", "428.0", time.Date(2006, time.March, 15, 15, 0, 0, 0, time.UTC)))
	es.Events = append(es.Events, conditionEvent("4", "Diabetes", "250.0", time.Date(2012, time.May, 15, 15, 0, 0, 0, time.UTC)))
	es.Events = append(es.Events, ageEvent("5", 75, time.Date(2015, time.July, 1, 0, 0, 0, 0, time.UTC)))
	results, err := cs.Plugin.Calculate(es, cs.FHIREndpointURL)

	c.Assert(err, NotNil)
	c.Assert(err, FitsTypeOf, plugin.NotApplicableError{})
	c.Assert(err.Error(), Equals, "CHA2DS2-VASc is only applicable to patients with Atrial Fibrillation")
	c.Assert(results, HasLen, 0)
}

func (cs *CHA2DS2VAScPluginSuite) assertResult(c *C, result plugin.RiskServiceCalculationResult, asOf time.Time, score int, pct float64, patientID string, chf, hypertension, diabetes, stroke, vasc, age, gender int) {
	c.Assert(result.AsOf, DeepEquals, asOf)
	c.Assert(*result.Score, Equals, score)
	c.Assert(*result.ProbabilityDecimal, Equals, pct)
	c.Assert(result.Pie, NotNil)
	pie := result.Pie
	c.Assert(pie.Patient, Equals, cs.FHIREndpointURL+"/Patient/"+patientID)
	c.Assert(pie.Slices, HasLen, 7)
	c.Assert(pie.Slices[0].Name, Equals, "Congestive Heart Failure")
	c.Assert(pie.Slices[0].Weight, Equals, 11)
	c.Assert(pie.Slices[0].MaxValue, Equals, 1)
	c.Assert(pie.Slices[0].Value, Equals, chf)
	c.Assert(pie.Slices[1].Name, Equals, "Hypertension")
	c.Assert(pie.Slices[1].Weight, Equals, 11)
	c.Assert(pie.Slices[1].MaxValue, Equals, 1)
	c.Assert(pie.Slices[1].Value, Equals, hypertension)
	c.Assert(pie.Slices[2].Name, Equals, "Diabetes")
	c.Assert(pie.Slices[2].Weight, Equals, 11)
	c.Assert(pie.Slices[2].MaxValue, Equals, 1)
	c.Assert(pie.Slices[2].Value, Equals, diabetes)
	c.Assert(pie.Slices[3].Name, Equals, "Stroke")
	c.Assert(pie.Slices[3].Weight, Equals, 22)
	c.Assert(pie.Slices[3].MaxValue, Equals, 2)
	c.Assert(pie.Slices[3].Value, Equals, stroke)
	c.Assert(pie.Slices[4].Name, Equals, "Vascular Disease")
	c.Assert(pie.Slices[4].Weight, Equals, 11)
	c.Assert(pie.Slices[4].MaxValue, Equals, 1)
	c.Assert(pie.Slices[4].Value, Equals, vasc)
	c.Assert(pie.Slices[5].Name, Equals, "Age")
	c.Assert(pie.Slices[5].Weight, Equals, 22)
	c.Assert(pie.Slices[5].MaxValue, Equals, 2)
	c.Assert(pie.Slices[5].Value, Equals, age)
	c.Assert(pie.Slices[6].Name, Equals, "Gender")
	c.Assert(pie.Slices[6].Weight, Equals, 11)
	c.Assert(pie.Slices[6].MaxValue, Equals, 1)
	c.Assert(pie.Slices[6].Value, Equals, gender)
}
