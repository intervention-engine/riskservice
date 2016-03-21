package plugin

import (
	"sort"
	"time"

	"github.com/intervention-engine/fhir/models"
)

type RiskServicePlugin interface {
	Config() RiskServicePluginConfig
	Calculate(es *EventStream, fhirEndpointURL string) ([]RiskServiceCalculationResult, error)
}

type RiskServicePluginConfig struct {
	Name                  string
	Method                models.CodeableConcept
	PredictedOutcome      models.CodeableConcept
	DefaultPieSlices      []Slice
	RequiredResourceTypes []string
	SignificantBirthdays  []int
}

type RiskServiceCalculationResult struct {
	AsOf               time.Time
	Score              *int
	ProbabilityDecimal *float64
	Pie                *Pie
}

// GetProbabilityDecimalOrScore returns the ProbabilityDecimal value if it exists, otherwise it returns the score.
// This approach preserves backwards compatibility with existing code, but should be reconsidered.
func (r *RiskServiceCalculationResult) GetProbabilityDecimalOrScore() *float64 {
	if r.ProbabilityDecimal != nil {
		return r.ProbabilityDecimal
	} else if r.Score != nil {
		f := float64(*r.Score)
		return &f
	}
	return nil
}

func (r *RiskServiceCalculationResult) ToRiskAssessment(patientId string, basisPieURL string, config RiskServicePluginConfig) *models.RiskAssessment {
	return &models.RiskAssessment{
		Subject: &models.Reference{Reference: "Patient/" + patientId},
		Method:  &config.Method,
		Date:    &models.FHIRDateTime{Time: r.AsOf, Precision: models.Timestamp},
		Prediction: []models.RiskAssessmentPredictionComponent{
			{
				ProbabilityDecimal: r.GetProbabilityDecimalOrScore(),
				Outcome:            &config.PredictedOutcome,
			},
		},
		Basis: []models.Reference{
			{Reference: basisPieURL + "/" + r.Pie.Id.Hex()},
		},
	}
}

// SortResultsByAsOfDate sorts the results by their as-of date
func SortResultsByAsOfDate(results []RiskServiceCalculationResult) {
	// Stable sort to preserve original order when dates are the same
	sort.Stable(byAsOfDate(results))
}

type byAsOfDate []RiskServiceCalculationResult

func (d byAsOfDate) Len() int {
	return len(d)
}
func (d byAsOfDate) Swap(i, j int) {
	d[i], d[j] = d[j], d[i]
}
func (d byAsOfDate) Less(i, j int) bool {
	return d[i].AsOf.Before(d[j].AsOf)
}

type NotApplicableError struct {
	msg string
}

func NewNotApplicableError(msg string) NotApplicableError {
	return NotApplicableError{msg: msg}
}

func (e NotApplicableError) Error() string { return e.msg }
