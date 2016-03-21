package plugin

import (
	"sort"
	"time"

	"github.com/intervention-engine/fhir/models"
)

// Event represents an event that may be of importance to a risk calculation
type Event struct {
	Date  time.Time
	Type  string
	End   bool
	Value interface{}
}

// EventStream represents a patient and an ordered stream of events
type EventStream struct {
	Patient *models.Patient
	Events  []Event
}

// NewEventStream creates a new EventStream for the given patient, initialized to 0 events
func NewEventStream(patient *models.Patient) *EventStream {
	es := new(EventStream)
	es.Patient = patient
	return es
}

// Clone does a shallow copy of the EventStream.  This is most helpful in case plugins
// modify the event stream.
func (es *EventStream) Clone() *EventStream {
	patient := new(models.Patient)
	if es.Patient != nil {
		*patient = *es.Patient
	}
	events := make([]Event, len(es.Events))
	copy(events, es.Events)
	return &EventStream{
		Patient: patient,
		Events:  events,
	}
}

// SortEventsByDate sorts the events by their date
func SortEventsByDate(events []Event) {
	// Stable sort to preserve original order when dates are the same
	sort.Stable(byEventDate(events))
}

type byEventDate []Event

func (d byEventDate) Len() int {
	return len(d)
}
func (d byEventDate) Swap(i, j int) {
	d[i], d[j] = d[j], d[i]
}
func (d byEventDate) Less(i, j int) bool {
	return d[i].Date.Before(d[j].Date)
}
