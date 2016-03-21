package plugin

import (
	"time"

	. "gopkg.in/check.v1"
)

type PieSuite struct {
	Pie *Pie
}

var _ = Suite(&PieSuite{})

func (p *PieSuite) SetUpTest(c *C) {
	p.Pie = NewPie("123")
	p.Pie.Slices = []Slice{
		{Name: "Cherry", Weight: 25, MaxValue: 2, Value: 1},
		{Name: "Apple", Weight: 75, MaxValue: 6, Value: 3},
	}
}

func (p *PieSuite) TestNewPie(c *C) {
	pie := NewPie("123")
	c.Assert(pie.Id.Hex(), Not(Equals), "")
	c.Assert(pie.Patient, Equals, "123")
	c.Assert(time.Since(pie.Created) < (1*time.Second), Equals, true)
	c.Assert(pie.Slices, HasLen, 0)
	c.Assert(pie.TotalValues(), Equals, 0)
}

func (p *PieSuite) TestTotalValues(c *C) {
	c.Assert(p.Pie.TotalValues(), Equals, 4)
}

func (p *PieSuite) TestUpdateSliceValue(c *C) {
	p.Pie.UpdateSliceValue("Apple", 5)
	c.Assert(p.Pie.Slices, DeepEquals, []Slice{
		{Name: "Cherry", Weight: 25, MaxValue: 2, Value: 1},
		{Name: "Apple", Weight: 75, MaxValue: 6, Value: 5},
	})
	c.Assert(p.Pie.TotalValues(), Equals, 6)
}

func (p *PieSuite) TestPieClone(c *C) {
	// Test initial clone
	clone := p.Pie.Clone(true)
	c.Assert(clone, Not(Equals), p.Pie)
	c.Assert(clone.Id.Hex(), Not(Equals), p.Pie.Id.Hex())
	c.Assert(clone.Created, Equals, p.Pie.Created)
	c.Assert(clone.Patient, Equals, p.Pie.Patient)
	c.Assert(&clone.Slices, Not(Equals), &p.Pie.Slices)
	c.Assert(clone.Slices, DeepEquals, p.Pie.Slices)

	// Modify clone and make sure it doesn't affect original
	clone.UpdateSliceValue("Apple", 2)
	c.Assert(clone.Slices[1].Value, Equals, 2)
	c.Assert(p.Pie.Slices[1].Value, Equals, 3)
}

func (p *PieSuite) TestPieCloneSameID(c *C) {
	// Test initial clone
	clone := p.Pie.Clone(false)
	c.Assert(clone.Id.Hex(), Equals, p.Pie.Id.Hex())
}
