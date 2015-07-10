package riskservice

import (
	"time"
)

// Structs in here represent the chart in Intervention
// Engine. Since the chart can't be represented in
// FHIR, the RiskAssessment basis will point back
// to one of these.

type Pie struct {
	Slices  []Slice
	Patient string
	Created time.Time
}

type Slice struct {
	Name   string
	Weight int
	Value  int
}

func NewPie(patientUrl string) *Pie {
	pie := &Pie{}
	pie.Patient = patientUrl
	pie.Created = time.Now()
	return pie
}

func (p *Pie) AddSlice(name string, weight, value int) {
	slice := Slice{Name: name, Weight: weight, Value: value}
	p.Slices = append(p.Slices, slice)
}
