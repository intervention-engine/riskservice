package assessments

import (
	"time"

	"github.com/intervention-engine/fhir/models"
	"github.com/intervention-engine/riskservice/plugin"
)

func ageEvent(id string, age int, effective time.Time) plugin.Event {
	return plugin.Event{
		Date:  effective,
		Type:  "Age",
		End:   false,
		Value: age,
	}
}

func conditionEvent(id, name, icd9Code string, onset time.Time) plugin.Event {
	condition := new(models.Condition)
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
		Date:  onset,
		Type:  "Condition",
		End:   false,
		Value: condition,
	}
}

func conditionStartAndEndEvents(id, name, icd9Code string, onset time.Time, abatement time.Time) (plugin.Event, plugin.Event) {
	start := conditionEvent(id, name, icd9Code, onset)
	condition := start.Value.(*models.Condition)
	condition.AbatementDateTime = &models.FHIRDateTime{Time: abatement, Precision: models.Timestamp}
	end := start
	end.Date = abatement
	end.End = true
	return start, end
}

func medicationEvent(id, name, rxNormCode string, activeDateTime time.Time) plugin.Event {
	medication := new(models.MedicationStatement)
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
		Date:  activeDateTime,
		Type:  "Medication",
		End:   false,
		Value: medication,
	}
}

func medicationStartAndEndEvents(id, name, rxNormCode string, active time.Time, inactive time.Time) (plugin.Event, plugin.Event) {
	start := medicationEvent(id, name, rxNormCode, active)
	medication := start.Value.(*models.MedicationStatement)
	medication.EffectivePeriod.End = &models.FHIRDateTime{Time: inactive, Precision: models.Timestamp}
	end := start
	end.Date = inactive
	end.End = true
	return start, end
}

func observationEvent(id, name, loincCode string, value models.Quantity, effective time.Time) plugin.Event {
	observation := new(models.Observation)
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
		Date:  effective,
		Type:  "Observation",
		End:   false,
		Value: observation,
	}
}

func encounterEvent(id, name, snomedCode string, start time.Time) plugin.Event {
	encounter := new(models.Encounter)
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
		Date:  start,
		Type:  "Encounter",
		End:   false,
		Value: encounter,
	}
}
