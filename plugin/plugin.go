package plugin

import (
	"net/url"
	"time"

	"github.com/intervention-engine/fhir/models"
	"github.com/intervention-engine/riskservice/assessment"
)

type RiskServicePlugin interface {
	Config() RiskServicePluginConfig
	Calculate(es *EventStream) ([]RiskServiceCalculationResult, error)
}

type RiskServicePluginConfig struct {
	Name                  string
	Method                models.CodeableConcept
	PredictedOutcome      models.CodeableConcept
	DefaultPieSlices      []assessment.Slice
	RequiredResourceTypes []string
	SignificantBirthdays  []int
}

type RiskServiceCalculationResult struct {
	AsOf               time.Time
	Score              *int
	ProbabilityDecimal *float64
	Pie                *assessment.Pie
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

func (r *RiskServiceCalculationResult) ToRiskAssessment(patientId string, basisPieURL url.URL, config RiskServicePluginConfig) *models.RiskAssessment {
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
			{Reference: basisPieURL.String() + "/" + r.Pie.Id.Hex()},
		},
	}
}

type NotApplicableError struct {
	msg string
}

func NewNotApplicableError(msg string) NotApplicableError {
	return NotApplicableError{msg: msg}
}

func (e NotApplicableError) Error() string { return e.msg }
