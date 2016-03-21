package plugin

import (
	"time"

	"gopkg.in/mgo.v2/bson"
)

// Pie represents the chart in Intervention Engine. Since the chart can't
// be represented in FHIR, the RiskAssessment basis will point back to
// one of these.
type Pie struct {
	Id      bson.ObjectId `bson:"_id" json:"id"`
	Slices  []Slice       `json:"slices"`
	Patient string        `json:"patient"`
	Created time.Time     `json:"created"`
}

// Slice represents a component that factors into the overall risk assessment
// algorithm.  In the chart, it appears as a slice in the pie.
type Slice struct {
	Name     string `json:"name"`
	Weight   int    `json:"weight"`
	Value    int    `json:"value"`
	MaxValue int    `json:"maxValue,omitempty"`
}

// NewPie constructs a new pie for the given patient, sets the Create time to
// now, and generates a new ID.  Slices are initially empty.
func NewPie(patientUrl string) *Pie {
	pie := &Pie{}
	pie.Patient = patientUrl
	pie.Created = time.Now()
	pie.Id = bson.NewObjectId()
	return pie
}

// Clone creates a copy of the pie.  If generateNewID is true, it will give
// the clone a new identity.  Slices of the clone can be modified without
// affecting the original.
func (p *Pie) Clone(generateNewID bool) *Pie {
	cloned := *p
	if generateNewID {
		cloned.Id = bson.NewObjectId()
	}
	cloned.Slices = make([]Slice, len(p.Slices))
	copy(cloned.Slices, p.Slices)
	return &cloned
}

// UpdateSliceValue is a convenience function that finds the slice with
// the given name and updates its value.
func (p *Pie) UpdateSliceValue(name string, value int) {
	for i := range p.Slices {
		if p.Slices[i].Name == name {
			p.Slices[i].Value = value
			return
		}
	}
}

// TotalValues sums up all the values in the slices.
func (p *Pie) TotalValues() int {
	total := 0
	for i := range p.Slices {
		total += p.Slices[i].Value
	}
	return total
}
