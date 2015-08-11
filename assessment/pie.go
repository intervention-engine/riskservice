package assessment

import (
	"gopkg.in/mgo.v2/bson"
	"time"
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

func (p *Pie) AddSlice(name string, weight int, value ...int) {
	slice := Slice{Name: name, Weight: weight, Value: value[0]}
	if len(value) == 2 {
		slice.MaxValue = value[1]
	}
	p.Slices = append(p.Slices, slice)
}
