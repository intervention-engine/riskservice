package simple

import (
	"testing"
	"time"

	"github.com/intervention-engine/fhir/models"
	"github.com/intervention-engine/riskservice/plugin"
	. "gopkg.in/check.v1"
)

type SimplePluginSuite struct {
	Plugin *SimplePlugin
}

func Test(t *testing.T) { TestingT(t) }

var _ = Suite(&SimplePluginSuite{})

func (cs *SimplePluginSuite) SetUpSuite(c *C) {
	cs.Plugin = &SimplePlugin{}
}

func (cs *SimplePluginSuite) TearDownSuite(c *C) {
	cs.Plugin = nil
}

func (cs *SimplePluginSuite) TestPatientWithNoConditionsAndNoMeds(c *C) {
	birthDate := &models.FHIRDateTime{Time: time.Date(1940, time.July, 1, 0, 0, 0, 0, time.UTC), Precision: models.Date}
	patient := &models.Patient{Gender: "female", BirthDate: birthDate}
	patient.Id = "1223"
	es := plugin.NewEventStream(patient)
	results, err := cs.Plugin.Calculate(es)
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
	es.AddEvent(conditionEvent("1", "Atrial Fibrillation", "427.31", time.Date(2010, time.February, 15, 15, 0, 0, 0, time.UTC)))
	es.AddEvent(conditionEvent("2", "Hypertension", "401.0", time.Date(2015, time.April, 15, 15, 0, 0, 0, time.UTC)))
	results, err := cs.Plugin.Calculate(es)
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
	es.AddEvent(medicationEvent("1", "Aspirin", "1191", time.Date(2010, time.April, 15, 15, 30, 0, 0, time.UTC)))
	es.AddEvent(medicationEvent("2", "Lisinopril", "104377", time.Date(2015, time.April, 15, 15, 30, 0, 0, time.UTC)))
	results, err := cs.Plugin.Calculate(es)
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
	es.AddEvent(conditionEvent("1", "Atrial Fibrillation", "427.31", time.Date(2010, time.February, 15, 15, 0, 0, 0, time.UTC)))
	es.AddEvent(medicationEvent("2", "Aspirin", "1191", time.Date(2010, time.February, 15, 15, 30, 0, 0, time.UTC)))
	es.AddEvent(conditionEvent("3", "Hypertension", "401.0", time.Date(2015, time.April, 15, 15, 0, 0, 0, time.UTC)))
	es.AddEvent(medicationEvent("4", "Lisinopril", "104377", time.Date(2015, time.April, 15, 15, 30, 0, 0, time.UTC)))
	results, err := cs.Plugin.Calculate(es)
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
	es.AddEvent(afibStart)
	es.AddEvent(aspirinStart)
	es.AddEvent(conditionEvent("3", "Hypertension", "401.0", time.Date(2015, time.April, 15, 15, 0, 0, 0, time.UTC)))
	es.AddEvent(medicationEvent("4", "Lisinopril", "104377", time.Date(2015, time.April, 15, 15, 30, 0, 0, time.UTC)))
	es.AddEvent(afibEnd)
	es.AddEvent(aspirinEnd)
	results, err := cs.Plugin.Calculate(es)
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
	es.AddEvent(conditionEvent("1", "Atrial Fibrillation", "427.31", time.Date(2010, time.February, 15, 15, 0, 0, 0, time.UTC)))
	es.AddEvent(medicationEvent("2", "Aspirin", "1191", time.Date(2010, time.February, 15, 15, 30, 0, 0, time.UTC)))
	es.AddEvent(conditionEvent("3", "Hypertension", "401.0", time.Date(2015, time.April, 15, 15, 0, 0, 0, time.UTC)))
	es.AddEvent(medicationEvent("4", "Lisinopril", "104377", time.Date(2015, time.April, 15, 15, 30, 0, 0, time.UTC)))
	es.AddEvent(conditionEvent("5", "Hypertension", "401.0", time.Date(2015, time.May, 1, 15, 0, 0, 0, time.UTC)))
	es.AddEvent(medicationEvent("6", "Lisinopril", "104377", time.Date(2015, time.May, 1, 15, 30, 0, 0, time.UTC)))

	results, err := cs.Plugin.Calculate(es)
	c.Assert(err, IsNil)
	c.Assert(results, HasLen, 6)
	cs.assertResult(c, results[0], time.Date(2010, time.February, 15, 15, 0, 0, 0, time.UTC), 1, "1223", 1, 0)
	cs.assertResult(c, results[1], time.Date(2010, time.February, 15, 15, 30, 0, 0, time.UTC), 2, "1223", 1, 1)
	cs.assertResult(c, results[2], time.Date(2015, time.April, 15, 15, 0, 0, 0, time.UTC), 3, "1223", 2, 1)
	cs.assertResult(c, results[3], time.Date(2015, time.April, 15, 15, 30, 0, 0, time.UTC), 4, "1223", 2, 2)
	cs.assertResult(c, results[4], time.Date(2015, time.May, 1, 15, 0, 0, 0, time.UTC), 4, "1223", 2, 2)
	cs.assertResult(c, results[5], time.Date(2015, time.May, 1, 15, 30, 0, 0, time.UTC), 4, "1223", 2, 2)
}

func (cs *SimplePluginSuite) TestNonSignificantEvents(c *C) {
	birthDate := &models.FHIRDateTime{Time: time.Date(1940, time.July, 1, 0, 0, 0, 0, time.UTC), Precision: models.Date}
	patient := &models.Patient{Gender: "female", BirthDate: birthDate}
	patient.Id = "1223"
	es := plugin.NewEventStream(patient)
	es.AddEvent(conditionEvent("1", "Atrial Fibrillation", "427.31", time.Date(2010, time.February, 15, 15, 0, 0, 0, time.UTC)))
	es.AddEvent(medicationEvent("2", "Aspirin", "1191", time.Date(2010, time.February, 15, 15, 30, 0, 0, time.UTC)))
	es.AddEvent(encounterEvent("3", "Consultation", "11429006", time.Date(2012, time.March, 15, 15, 0, 0, 0, time.UTC)))
	weightFloat := float64(163)
	weight := models.Quantity{Value: &weightFloat, Unit: "lb_av"}
	es.AddEvent(observationEvent("4", "Body Weight", "29463-7", weight, time.Date(2012, time.March, 15, 15, 30, 0, 0, time.UTC)))
	es.AddEvent(conditionEvent("5", "Hypertension", "401.0", time.Date(2015, time.April, 15, 15, 0, 0, 0, time.UTC)))
	es.AddEvent(medicationEvent("6", "Lisinopril", "104377", time.Date(2015, time.April, 15, 15, 30, 0, 0, time.UTC)))
	results, err := cs.Plugin.Calculate(es)
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
	c.Assert(pie.Patient, Equals, "Patient/"+patientID)
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

func conditionEvent(id, name, icd9Code string, onset time.Time) plugin.Event {
	var condition models.Condition
	condition.Id = id
	condition.Code = &models.CodeableConcept{
		Coding: []models.Coding{
			models.Coding{System: "http://hl7.org/fhir/sid/icd-9", Code: icd9Code, Display: name},
		},
		Text: name,
	}
	condition.OnsetDateTime = &models.FHIRDateTime{Time: onset, Precision: models.Timestamp}
	condition.VerificationStatus = "confirmed"

	return plugin.Event{
		Date:     onset,
		Type:     "Condition",
		End:      false,
		Resource: condition,
	}
}

func conditionStartAndEndEvents(id, name, icd9Code string, onset time.Time, abatement time.Time) (plugin.Event, plugin.Event) {
	start := conditionEvent(id, name, icd9Code, onset)
	condition := start.Resource.(models.Condition)
	condition.AbatementDateTime = &models.FHIRDateTime{Time: abatement, Precision: models.Timestamp}
	end := start
	end.Date = abatement
	end.End = true
	return start, end
}

func medicationEvent(id, name, rxNormCode string, activeDateTime time.Time) plugin.Event {
	var medication models.MedicationStatement
	medication.Id = id
	medication.MedicationCodeableConcept = &models.CodeableConcept{
		Coding: []models.Coding{
			models.Coding{System: "http://www.nlm.nih.gov/research/umls/rxnorm/", Code: rxNormCode, Display: name},
		},
		Text: name,
	}
	medication.EffectivePeriod = &models.Period{
		Start: &models.FHIRDateTime{Time: activeDateTime, Precision: models.Timestamp},
	}
	medication.Status = "active"

	return plugin.Event{
		Date:     activeDateTime,
		Type:     "Medication",
		End:      false,
		Resource: medication,
	}
}

func medicationStartAndEndEvents(id, name, rxNormCode string, active time.Time, inactive time.Time) (plugin.Event, plugin.Event) {
	start := medicationEvent(id, name, rxNormCode, active)
	medication := start.Resource.(models.MedicationStatement)
	medication.EffectivePeriod.End = &models.FHIRDateTime{Time: inactive, Precision: models.Timestamp}
	end := start
	end.Date = inactive
	end.End = true
	return start, end
}

func observationEvent(id, name, loincCode string, value models.Quantity, effective time.Time) plugin.Event {
	var observation models.Observation
	observation.Id = id
	observation.Code = &models.CodeableConcept{
		Coding: []models.Coding{
			models.Coding{System: "http://loinc.org", Code: loincCode, Display: name},
		},
		Text: name,
	}
	observation.ValueQuantity = &value
	observation.EffectiveDateTime = &models.FHIRDateTime{Time: effective, Precision: models.Timestamp}
	observation.Status = "final"

	return plugin.Event{
		Date:     effective,
		Type:     "Observation",
		End:      false,
		Resource: observation,
	}
}

func encounterEvent(id, name, snomedCode string, start time.Time) plugin.Event {
	var encounter models.Encounter
	encounter.Id = id
	encounter.Type = []models.CodeableConcept{
		{
			Coding: []models.Coding{
				models.Coding{System: "http://snomed.info/sct", Code: snomedCode, Display: name},
			},
			Text: name,
		},
	}
	encounter.Period = &models.Period{
		Start: &models.FHIRDateTime{Time: start, Precision: models.Timestamp},
	}
	encounter.Status = "finished"

	return plugin.Event{
		Date:     start,
		Type:     "Encounter",
		End:      false,
		Resource: encounter,
	}
}
