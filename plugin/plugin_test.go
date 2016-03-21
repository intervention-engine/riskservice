package plugin

import (
	"sort"
	"testing"
	"time"

	"github.com/intervention-engine/fhir/models"

	. "gopkg.in/check.v1"
)

type PluginSuite struct{}

func Test(t *testing.T) { TestingT(t) }

var _ = Suite(&PluginSuite{})

func (p *PluginSuite) TestGetProbabilityDecimalOrScore(c *C) {
	tests := []struct {
		score   *int
		decimal *float64
		expect  *float64
	}{
		{nil, ptrToFlt(45.6), ptrToFlt(45.6)},
		{ptrToInt(123), nil, ptrToFlt(float64(123))},
		{ptrToInt(123), ptrToFlt(45.6), ptrToFlt(45.6)},
		{nil, nil, nil},
	}

	for _, t := range tests {
		result := RiskServiceCalculationResult{
			Score:              t.score,
			ProbabilityDecimal: t.decimal,
		}
		c.Assert(result.GetProbabilityDecimalOrScore(), DeepEquals, t.expect)
	}
}

func (p *PluginSuite) TestToRiskAssessment(c *C) {
	myConfig := RiskServicePluginConfig{
		Name: "Test Risk Assessment",
		Method: models.CodeableConcept{
			Coding: []models.Coding{{System: "http://interventionengine.org/risk-assessments", Code: "Simple"}},
			Text:   "Test Risk Assessment",
		},
		PredictedOutcome: models.CodeableConcept{Text: "Something Bad"},
		DefaultPieSlices: []Slice{
			{Name: "Cherry", Weight: 25, MaxValue: 2},
			{Name: "Apple", Weight: 75, MaxValue: 6},
		},
		RequiredResourceTypes: []string{"Condition"},
	}

	// Test it with probabilityDecimal and a score
	result := RiskServiceCalculationResult{
		AsOf:               time.Now(),
		Score:              ptrToInt(123),
		ProbabilityDecimal: ptrToFlt(45.6),
		Pie:                NewPie("http://example.org/Patient/abc"),
	}
	ra := result.ToRiskAssessment("abc", "http://foo.org/pie", myConfig)
	expected := &models.RiskAssessment{
		Subject: &models.Reference{Reference: "Patient/abc"},
		Method: &models.CodeableConcept{
			Coding: []models.Coding{{System: "http://interventionengine.org/risk-assessments", Code: "Simple"}},
			Text:   "Test Risk Assessment",
		},
		Date: &models.FHIRDateTime{Time: result.AsOf, Precision: models.Timestamp},
		Prediction: []models.RiskAssessmentPredictionComponent{
			{
				ProbabilityDecimal: ptrToFlt(45.6),
				Outcome:            &models.CodeableConcept{Text: "Something Bad"},
			},
		},
		Basis: []models.Reference{
			{Reference: "http://foo.org/pie/" + result.Pie.Id.Hex()},
		},
	}
	c.Assert(ra, DeepEquals, expected)

	// Now test it with just a score
	result.ProbabilityDecimal = nil
	ra = result.ToRiskAssessment("abc", "http://foo.org/pie", myConfig)
	expected.Prediction[0].ProbabilityDecimal = ptrToFlt(float64(123))
	c.Assert(ra, DeepEquals, expected)
}

func (p *PluginSuite) TestSortByAsOf(c *C) {
	results := []RiskServiceCalculationResult{
		{
			AsOf:  time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
			Score: ptrToInt(1),
		},
		{
			AsOf:  time.Date(2004, time.January, 1, 0, 0, 0, 0, time.UTC),
			Score: ptrToInt(2),
		},
		{
			AsOf:  time.Date(2002, time.January, 1, 0, 0, 0, 0, time.UTC),
			Score: ptrToInt(3),
		},
		{
			AsOf:  time.Date(2005, time.January, 1, 0, 0, 0, 0, time.UTC),
			Score: ptrToInt(4),
		},
		{
			AsOf:  time.Date(1999, time.January, 1, 0, 0, 0, 0, time.UTC),
			Score: ptrToInt(5),
		},
	}

	sort.Sort(byAsOfDate(results))
	for i, score := range []int{5, 1, 3, 2, 4} {
		c.Assert(*results[i].Score, Equals, score)
	}
}

func (p *PluginSuite) TestNewNotApplicableError(c *C) {
	err := NewNotApplicableError("Foo is not applicable")
	c.Assert(err.Error(), Equals, "Foo is not applicable")
}

func ptrToInt(i int) *int {
	return &i
}

func ptrToFlt(f float64) *float64 {
	return &f
}
