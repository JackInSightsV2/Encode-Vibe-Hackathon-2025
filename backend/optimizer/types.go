package optimizer

import (
	"time"
)

// ExperimentStatus represents the current state of an experiment
type ExperimentStatus string

const (
	ExperimentStatusPending   ExperimentStatus = "pending"
	ExperimentStatusRunning   ExperimentStatus = "running"
	ExperimentStatusCompleted ExperimentStatus = "completed"
	ExperimentStatusStopped   ExperimentStatus = "stopped"
	ExperimentStatusFailed    ExperimentStatus = "failed"
)

// Experiment represents an A/B testing experiment for rules
type Experiment struct {
	ID         string           `json:"id"`
	Name       string           `json:"name"`
	Status     ExperimentStatus `json:"status"`
	StartTime  time.Time        `json:"start_time"`
	EndTime    *time.Time       `json:"end_time,omitempty"`
	StopReason string           `json:"stop_reason,omitempty"`
	
	// Opik integration
	OpikID string `json:"opik_id,omitempty"`
	
	// Experiment configuration
	Objectives []Objective     `json:"objectives"`
	Control    Rule            `json:"control"`
	Variants   []Rule          `json:"variants"`
	Traffic    TrafficSplit    `json:"traffic"`
	Config     ExperimentConfig `json:"config"`
	
	// Results
	Results *ExperimentResults `json:"results"`
	
	// Internal
	opikClient interface{} `json:"-"`
}

// ExperimentConfig contains configuration for creating an experiment
type ExperimentConfig struct {
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Objectives  []Objective  `json:"objectives"`
	Control     Rule         `json:"control"`
	Variants    []Rule       `json:"variants"`
	Traffic     TrafficSplit `json:"traffic"`
	Duration    time.Duration `json:"duration"`
	MinSampleSize int        `json:"min_sample_size"`
	
	// Early stopping criteria
	EarlyStoppingEnabled bool    `json:"early_stopping_enabled"`
	MinDetectionImprovement float64 `json:"min_detection_improvement"`
	MaxDegradationTolerance float64 `json:"max_degradation_tolerance"`
}

// TrafficSplit defines how traffic is distributed across variants
type TrafficSplit struct {
	Control  float64            `json:"control"`
	Variants map[string]float64 `json:"variants"`
}

// Rule represents a moderation rule that can be optimized
type Rule struct {
	ID         string                 `json:"id"`
	Name       string                 `json:"name"`
	Type       RuleType               `json:"type"`
	Parameters map[string]interface{} `json:"parameters"`
	Enabled    bool                   `json:"enabled"`
	Priority   int                    `json:"priority"`
	
	// Metadata
	Description string            `json:"description"`
	Tags        []string          `json:"tags"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
	Version     string            `json:"version"`
	ParentID    string            `json:"parent_id,omitempty"`
	
	// Performance metrics
	LastEvaluated *time.Time `json:"last_evaluated,omitempty"`
	Fitness       float64    `json:"fitness"`
}

// RuleType represents the type of rule
type RuleType string

const (
	RuleTypeRegex    RuleType = "regex"
	RuleTypeSemantic RuleType = "semantic"
	RuleTypeHybrid   RuleType = "hybrid"
	RuleTypePII      RuleType = "pii"
	RuleTypeCustom   RuleType = "custom"
)

// Clone creates a deep copy of a rule
func (r *Rule) Clone() Rule {
	clone := *r
	
	// Deep copy parameters
	clone.Parameters = make(map[string]interface{})
	for k, v := range r.Parameters {
		clone.Parameters[k] = v
	}
	
	// Deep copy tags
	if r.Tags != nil {
		clone.Tags = make([]string, len(r.Tags))
		copy(clone.Tags, r.Tags)
	}
	
	clone.CreatedAt = time.Now()
	clone.UpdatedAt = time.Now()
	
	return clone
}

// GetMutableParameters returns parameters that can be mutated
func (r *Rule) GetMutableParameters() []Parameter {
	var params []Parameter
	
	switch r.Type {
	case RuleTypeRegex:
		if pattern, ok := r.Parameters["pattern"].(string); ok {
			params = append(params, Parameter{
				Name:    "pattern",
				Type:    ParameterTypeString,
				Current: pattern,
			})
		}
		if flags, ok := r.Parameters["flags"].(string); ok {
			params = append(params, Parameter{
				Name:    "flags",
				Type:    ParameterTypeString,
				Current: flags,
			})
		}
	case RuleTypeSemantic:
		if threshold, ok := r.Parameters["threshold"].(float64); ok {
			params = append(params, Parameter{
				Name:    "threshold",
				Type:    ParameterTypeFloat,
				Min:     0.0,
				Max:     1.0,
				Current: threshold,
			})
		}
		if model, ok := r.Parameters["model"].(string); ok {
			params = append(params, Parameter{
				Name:    "model",
				Type:    ParameterTypeEnum,
				Options: []interface{}{"bert", "roberta", "distilbert"},
				Current: model,
			})
		}
	case RuleTypePII:
		if types, ok := r.Parameters["pii_types"].([]string); ok {
			params = append(params, Parameter{
				Name:    "pii_types",
				Type:    ParameterTypeStringArray,
				Current: types,
			})
		}
		if sensitivity, ok := r.Parameters["sensitivity"].(float64); ok {
			params = append(params, Parameter{
				Name:    "sensitivity",
				Type:    ParameterTypeFloat,
				Min:     0.0,
				Max:     1.0,
				Current: sensitivity,
			})
		}
	}
	
	return params
}

// SetParameter sets a parameter value with type checking
func (r *Rule) SetParameter(name string, value interface{}) error {
	if r.Parameters == nil {
		r.Parameters = make(map[string]interface{})
	}
	
	r.Parameters[name] = value
	r.UpdatedAt = time.Now()
	
	return nil
}

// Parameter represents a configurable parameter of a rule
type Parameter struct {
	Name     string        `json:"name"`
	Type     ParameterType `json:"type"`
	Min      interface{}   `json:"min,omitempty"`
	Max      interface{}   `json:"max,omitempty"`
	Options  []interface{} `json:"options,omitempty"`
	Current  interface{}   `json:"current"`
}

// ParameterType represents the type of a parameter
type ParameterType string

const (
	ParameterTypeString      ParameterType = "string"
	ParameterTypeInt         ParameterType = "int"
	ParameterTypeFloat       ParameterType = "float"
	ParameterTypeBool        ParameterType = "bool"
	ParameterTypeEnum        ParameterType = "enum"
	ParameterTypeStringArray ParameterType = "string_array"
)

// ExperimentResults contains the results of an experiment
type ExperimentResults struct {
	VariantMetrics map[string]*VariantMetrics `json:"variant_metrics"`
	Winner         *VariantMetrics            `json:"winner,omitempty"`
	Significance   *SignificanceTest          `json:"significance,omitempty"`
	Summary        string                     `json:"summary"`
	
	// Detailed analysis
	ObjectiveAnalysis map[string]*ObjectiveAnalysis `json:"objective_analysis"`
	RecommendedAction string                        `json:"recommended_action"`
	ConfidenceLevel   float64                       `json:"confidence_level"`
	ValidationErrors  []string                      `json:"validation_errors"`
}

// NewExperimentResults creates a new experiment results instance
func NewExperimentResults() *ExperimentResults {
	return &ExperimentResults{
		VariantMetrics:    make(map[string]*VariantMetrics),
		ObjectiveAnalysis: make(map[string]*ObjectiveAnalysis),
		ValidationErrors:  make([]string, 0),
	}
}

// GetVariantMetrics retrieves metrics for a variant
func (er *ExperimentResults) GetVariantMetrics(variantID string) *VariantMetrics {
	return er.VariantMetrics[variantID]
}

// SetVariantMetrics sets metrics for a variant
func (er *ExperimentResults) SetVariantMetrics(variantID string, metrics *VariantMetrics) {
	if er.VariantMetrics == nil {
		er.VariantMetrics = make(map[string]*VariantMetrics)
	}
	er.VariantMetrics[variantID] = metrics
}

// GetBestVariant returns the variant with the highest fitness score
func (er *ExperimentResults) GetBestVariant() *VariantMetrics {
	var best *VariantMetrics
	for _, metrics := range er.VariantMetrics {
		if best == nil || metrics.FitnessScore > best.FitnessScore {
			best = metrics
		}
	}
	return best
}

// VariantMetrics contains performance metrics for a rule variant
type VariantMetrics struct {
	VariantID      string        `json:"variant_id"`
	SampleSize     int           `json:"sample_size"`
	DetectionRate  float64       `json:"detection_rate"`
	FalsePositiveRate float64    `json:"false_positive_rate"`
	FalseNegativeRate float64    `json:"false_negative_rate"`
	UserSatisfaction  float64    `json:"user_satisfaction"`
	AvgResponseTime   time.Duration `json:"avg_response_time"`
	Throughput        float64    `json:"throughput"`
	ErrorRate         float64    `json:"error_rate"`
	
	// Calculated metrics
	FitnessScore   float64 `json:"fitness_score"`
	Precision      float64 `json:"precision"`
	Recall         float64 `json:"recall"`
	F1Score        float64 `json:"f1_score"`
	
	// Statistical data
	Successes      int   `json:"successes"`
	Failures       int   `json:"failures"`
	ResponseTimes  []int64 `json:"response_times,omitempty"`
	
	// Timestamps
	FirstSample time.Time `json:"first_sample"`
	LastSample  time.Time `json:"last_sample"`
}

// UpdateMetrics updates the variant metrics with new data
func (vm *VariantMetrics) UpdateMetrics(detection, falsePositive bool, responseTime time.Duration) {
	vm.SampleSize++
	
	if detection {
		vm.Successes++
	} else {
		vm.Failures++
	}
	
	// Update rates
	vm.DetectionRate = float64(vm.Successes) / float64(vm.SampleSize)
	if falsePositive {
		vm.FalsePositiveRate = (vm.FalsePositiveRate*float64(vm.SampleSize-1) + 1.0) / float64(vm.SampleSize)
	} else {
		vm.FalsePositiveRate = (vm.FalsePositiveRate * float64(vm.SampleSize-1)) / float64(vm.SampleSize)
	}
	
	// Update response time
	if vm.SampleSize == 1 {
		vm.AvgResponseTime = responseTime
	} else {
		totalTime := vm.AvgResponseTime * time.Duration(vm.SampleSize-1)
		vm.AvgResponseTime = (totalTime + responseTime) / time.Duration(vm.SampleSize)
	}
	
	vm.LastSample = time.Now()
	if vm.SampleSize == 1 {
		vm.FirstSample = vm.LastSample
	}
}

// SignificanceTest contains statistical significance test results
type SignificanceTest struct {
	PValue      float64 `json:"p_value"`
	ZScore      float64 `json:"z_score"`
	Significant bool    `json:"significant"`
	Improvement float64 `json:"improvement_percentage"`
	Method      string  `json:"method"`
}

// ObjectiveAnalysis contains analysis for a specific objective
type ObjectiveAnalysis struct {
	ObjectiveName string  `json:"objective_name"`
	ControlValue  float64 `json:"control_value"`
	VariantValue  float64 `json:"variant_value"`
	Improvement   float64 `json:"improvement_percentage"`
	Significant   bool    `json:"significant"`
	MeetsTarget   bool    `json:"meets_target"`
}

// GetDuration returns the duration of the experiment
func (e *Experiment) GetDuration() time.Duration {
	if e.EndTime != nil {
		return e.EndTime.Sub(e.StartTime)
	}
	return time.Since(e.StartTime)
}

// IsActive returns true if the experiment is currently running
func (e *Experiment) IsActive() bool {
	return e.Status == ExperimentStatusRunning
}

// GetVariantByID returns a variant by its ID
func (e *Experiment) GetVariantByID(id string) (*Rule, bool) {
	if id == "control" {
		return &e.Control, true
	}
	
	for _, variant := range e.Variants {
		if variant.ID == id {
			return &variant, true
		}
	}
	
	return nil, false
}