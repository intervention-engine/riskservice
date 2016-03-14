package plugin

import (
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/intervention-engine/fhir/models"
)

// Event represents an event that may be of importance to a risk calculation
type Event struct {
	Date     time.Time
	Type     string
	End      bool
	Resource interface{}
}

// EventStream represents a patient and an ordered stream of events
type EventStream struct {
	Patient *models.Patient
	Events  []Event
}

// NewEventStream creates a new EventStream for the given patient, initialized to 0 events
func NewEventStream(patient *models.Patient) *EventStream {
	es := EventStream{}
	es.Patient = patient
	es.Events = make([]Event, 0)
	return &es
}

// AddEvent is a convenience function for adding an event to the EventStream
func (es *EventStream) AddEvent(e Event) {
	es.Events = append(es.Events, e)
}

// ResourcesToEventStream takes a slice of FHIR resources and converts them to an EventStream.  Currently only a
// limited set of resource types are supported, with unsupported resource types resulting in an error.  If the
// slice of resources contains more than one patient, this is also considered an error.
func ResourcesToEventStream(resources []interface{}) (es *EventStream, err error) {
	es = NewEventStream(nil)
	for _, r := range resources {
		switch r := r.(type) {
		default:
			err = fmt.Errorf("Unsupported: Converting %s to Event", reflect.TypeOf(r).Name())
			return
		case models.Patient:
			if es.Patient != nil {
				err = errors.New("Found more than one patient in resources")
				return
			}
		case models.Condition:
			if onset, err := findDate(false, r.OnsetDateTime, r.OnsetPeriod, r.DateRecorded); err != nil {
				es.AddEvent(Event{Date: onset, Type: "Condition", End: false, Resource: r})
			}
			if abatement, err := findDate(true, r.AbatementDateTime, r.AbatementPeriod); err != nil {
				es.AddEvent(Event{Date: abatement, Type: "Condition", End: true, Resource: r})
			}
			// TODO: What happens if there is no date at all?
		case models.MedicationStatement:
			if active, err := findDate(false, r.EffectiveDateTime, r.EffectivePeriod, r.DateAsserted); err != nil {
				es.AddEvent(Event{Date: active, Type: "Medication", End: false, Resource: r})
			}
			if inactive, err := findDate(true, r.EffectivePeriod); err != nil {
				es.AddEvent(Event{Date: inactive, Type: "Medication", End: true, Resource: r})
			}
			// TODO: What happens if there is no date at all?
		case models.Observation:
			if effective, err := findDate(false, r.EffectiveDateTime, r.EffectivePeriod, r.Issued); err != nil {
				es.AddEvent(Event{Date: effective, Type: "Observation", End: false, Resource: r})
			}
			if ineffective, err := findDate(true, r.EffectivePeriod); err != nil {
				es.AddEvent(Event{Date: ineffective, Type: "Observation", End: true, Resource: r})
			}
			// TODO: What happens if there is no date at all?
		}
	}
	return
}

func BundleToEventStream(bundle *models.Bundle) (*EventStream, error) {
	resources := make([]interface{}, len(bundle.Entry))
	for i := range bundle.Entry {
		resources[i] = bundle.Entry[i].Resource
	}
	return ResourcesToEventStream(resources)
}

func getBirthdayObservations(patient *models.Patient, ages ...int) (birthdays []models.Observation, err error) {
	if patient.BirthDate == nil {
		err = errors.New("Unknown birthday")
		return
	}

	for _, age := range ages {
		bd := patient.BirthDate.Time.AddDate(age, 0, 0)
		if bd.Before(time.Now()) {
			ageFloat := float64(age)
			obs := models.Observation{
				// TODO: Fix code and unit
				Code: &models.CodeableConcept{
					Coding: []models.Coding{
						{System: "http://loinc.org", Code: "30525-0"},
					},
					Text: "Age",
				},
				ValueQuantity: &models.Quantity{
					Value: &ageFloat,
					Unit:  "a",
				},
				EffectiveDateTime: &models.FHIRDateTime{Time: bd, Precision: models.Precision(models.Date)},
				Status:            "final",
			}
			birthdays = append(birthdays, obs)
		}
	}
	return
}

func findDate(usePeriodEnd bool, datesAndPeriods ...interface{}) (time.Time, error) {
	for _, t := range datesAndPeriods {
		switch t := t.(type) {
		case models.FHIRDateTime:
			return t.Time, nil
		case *models.FHIRDateTime:
			if t != nil {
				return t.Time, nil
			}
		case models.Period:
			if !usePeriodEnd && t.Start != nil {
				return t.Start.Time, nil
			} else if usePeriodEnd && t.End != nil {
				return t.End.Time, nil
			}
		case *models.Period:
			if !usePeriodEnd && t != nil && t.Start != nil {
				return t.Start.Time, nil
			} else if usePeriodEnd && t != nil && t.End != nil {
				return t.End.Time, nil
			}
		}
	}

	return time.Time{}, errors.New("No date found")
}
