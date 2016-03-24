package plugin

import (
	"time"

	"github.com/intervention-engine/fhir/models"
	. "gopkg.in/check.v1"
)

type EventsSuite struct {
}

var _ = Suite(&EventsSuite{})

func (p *EventsSuite) TestNewEventStream(c *C) {
	patient := new(models.Patient)
	patient.Id = "123"

	es := NewEventStream(patient)
	c.Assert(es.Patient, DeepEquals, patient)
	c.Assert(es.Events, HasLen, 0)
}

func (p *PieSuite) TestEventStreamClone(c *C) {
	patient := new(models.Patient)
	patient.Id = "123"

	es := NewEventStream(patient)
	es.Events = []Event{
		{
			Date:  time.Now(),
			Type:  "Foo",
			End:   false,
			Value: 123,
		},
		{
			Date:  time.Now(),
			Type:  "Bar",
			End:   false,
			Value: 456,
		},
	}

	// Test initial clone
	clone := es.Clone()
	c.Assert(clone.Patient, DeepEquals, es.Patient)
	c.Assert(&clone.Events, Not(Equals), &es.Events)
	c.Assert(clone.Events, DeepEquals, es.Events)

	// Modify clone and make sure it doesn't affect original
	clone.Events[1].End = true
	c.Assert(es.Events[1].End, Equals, false)
}
