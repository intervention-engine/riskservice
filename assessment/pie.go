package assessment

import (
	"time"

	"gopkg.in/mgo.v2/bson"
)

// Structs in here represent the chart in Intervention
// Engine. Since the chart can't be represented in
// FHIR, the RiskAssessment basis will point back
// to one of these.

type Pie struct {
	Id      bson.ObjectId `bson:"_id" json:"id"`
	Slices  []Slice       `json:"slices"`
	Patient string        `json:"patient"`
	Created time.Time     `json:"created"`
}

type Slice struct {
	Name     string `json:"name"`
	Weight   int    `json:"weight"`
	Value    int    `json:"value"`
	MaxValue int    `json:"maxValue,omitempty"`
}

func NewPie(patientUrl string) *Pie {
	pie := &Pie{}
	pie.Patient = patientUrl
	pie.Created = time.Now()
	pie.Id = bson.NewObjectId()
	return pie
}

func (p *Pie) Clone(generateNewID bool) *Pie {
	cloned := *p
	if generateNewID {
		cloned.Id = bson.NewObjectId()
	}
	cloned.Slices = make([]Slice, len(p.Slices))
	copy(cloned.Slices, p.Slices)
	return &cloned
}

func (p *Pie) AddSlice(name string, weight int, value ...int) {
	slice := Slice{Name: name, Weight: weight, Value: value[0]}
	if len(value) == 2 {
		slice.MaxValue = value[1]
	}
	p.Slices = append(p.Slices, slice)
}

func (p *Pie) UpdateSliceValue(name string, value int) {
	for i := range p.Slices {
		if p.Slices[i].Name == name {
			p.Slices[i].Value = value
			return
		}
	}
}

func (p *Pie) TotalValues() int {
	total := 0
	for i := range p.Slices {
		total += p.Slices[i].Value
	}
	return total
}
