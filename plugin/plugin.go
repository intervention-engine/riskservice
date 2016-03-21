package plugin

import (
	"sort"
	"time"

	"github.com/intervention-engine/fhir/models"
)

// RiskServicePlugin provides the interface that risk service plugins should
// adhere to.  This pertains only to this implementation -- as the only real
// interface that matters is the FHIR API.  But... this provides an easy
// entry point for Go-based risk services.
type RiskServicePlugin interface {
	// Config returns the configuration information for the risk service plugin
	Config() RiskServicePluginConfig
	// Calculate accepts an EventStream and returns the slice of
	// RiskServiceCalculationResults that corresponds to the event stream.  This
	// results slice represents risks over time, with the last element being the
	// most recent risk assessment.
	Calculate(es *EventStream, fhirEndpointURL string) ([]RiskServiceCalculationResult, error)
}

// RiskServicePluginConfig represents key information about the risk service plugin.
type RiskServicePluginConfig struct {
	Name                  string
	Method                models.CodeableConcept
	PredictedOutcome      models.CodeableConcept
	DefaultPieSlices      []Slice
	RequiredResourceTypes []string
	SignificantBirthdays  []int
}

// RiskServiceCalculationResult represents risk assessment info for a given point
// in time.  The Score indicates a raw score from the algorithm (if applicable),
// while the ProbabilityDecimal represents a percentage probability of the predicted
// outcome.  Since it is a percentage, the value should never exceed 100.
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

// ToRiskAssessment converts the RiskServiceCalculationResult to a FHIR RiskAssessment.
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

// NotApplicableError indicates that the given algorithm is not applicable
// for the requested patient.  It would be inappropriate to return a score.
type NotApplicableError struct {
	msg string
}

// NewNotApplicableError returns a new NotApplicableError with the given
// message.
func NewNotApplicableError(msg string) NotApplicableError {
	return NotApplicableError{msg: msg}
}

func (e NotApplicableError) Error() string { return e.msg }
