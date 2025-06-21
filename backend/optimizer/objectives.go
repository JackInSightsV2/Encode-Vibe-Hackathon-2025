package optimizer

import (
	"fmt"
	"time"
)

// Objective represents an optimization objective
type Objective struct {
	Name        string        `json:"name" yaml:"name"`
	Weight      float64       `json:"weight" yaml:"weight"`
	Target      float64       `json:"target" yaml:"target"`
	Type        ObjectiveType `json:"type" yaml:"type"`
	Description string        `json:"description" yaml:"description"`
	Unit        string        `json:"unit" yaml:"unit"`
}

// ObjectiveType defines whether to minimize or maximize an objective
type ObjectiveType string

const (
	Minimize ObjectiveType = "minimize"
	Maximize ObjectiveType = "maximize"
)

// ObjectiveSet represents a collection of objectives with validation
type ObjectiveSet struct {
	Objectives []Objective `json:"objectives"`
	totalWeight float64
}

// GetDefaultObjectives returns the standard optimization objectives
func GetDefaultObjectives() []Objective {
	return []Objective{
		{
			Name:        "detection_rate",
			Weight:      0.4,
			Target:      0.95,
			Type:        Maximize,
			Description: "Percentage of actual threats correctly identified",
			Unit:        "percentage",
		},
		{
			Name:        "false_positive_rate",
			Weight:      0.3,
			Target:      0.05,
			Type:        Minimize,
			Description: "Percentage of benign content incorrectly flagged",
			Unit:        "percentage",
		},
		{
			Name:        "response_time_ms",
			Weight:      0.2,
			Target:      10.0,
			Type:        Minimize,
			Description: "Average time to process a request",
			Unit:        "milliseconds",
		},
		{
			Name:        "user_satisfaction",
			Weight:      0.1,
			Target:      0.9,
			Type:        Maximize,
			Description: "User satisfaction score based on feedback",
			Unit:        "score",
		},
	}
}

// GetSecurityFocusedObjectives returns objectives prioritizing security over speed
func GetSecurityFocusedObjectives() []Objective {
	return []Objective{
		{
			Name:        "detection_rate",
			Weight:      0.5,
			Target:      0.98,
			Type:        Maximize,
			Description: "High-priority threat detection",
			Unit:        "percentage",
		},
		{
			Name:        "false_negative_rate",
			Weight:      0.3,
			Target:      0.02,
			Type:        Minimize,
			Description: "Missed threats that should have been caught",
			Unit:        "percentage",
		},
		{
			Name:        "false_positive_rate",
			Weight:      0.15,
			Target:      0.08,
			Type:        Minimize,
			Description: "Acceptable higher false positive rate for security",
			Unit:        "percentage",
		},
		{
			Name:        "response_time_ms",
			Weight:      0.05,
			Target:      25.0,
			Type:        Minimize,
			Description: "Response time is less critical",
			Unit:        "milliseconds",
		},
	}
}

// GetPerformanceFocusedObjectives returns objectives prioritizing speed over security
func GetPerformanceFocusedObjectives() []Objective {
	return []Objective{
		{
			Name:        "response_time_ms",
			Weight:      0.4,
			Target:      5.0,
			Type:        Minimize,
			Description: "Ultra-fast response time",
			Unit:        "milliseconds",
		},
		{
			Name:        "throughput",
			Weight:      0.3,
			Target:      10000.0,
			Type:        Maximize,
			Description: "Requests processed per second",
			Unit:        "rps",
		},
		{
			Name:        "detection_rate",
			Weight:      0.2,
			Target:      0.90,
			Type:        Maximize,
			Description: "Acceptable detection rate for speed",
			Unit:        "percentage",
		},
		{
			Name:        "false_positive_rate",
			Weight:      0.1,
			Target:      0.1,
			Type:        Minimize,
			Description: "Moderate false positive tolerance",
			Unit:        "percentage",
		},
	}
}

// GetBalancedObjectives returns objectives balancing all factors
func GetBalancedObjectives() []Objective {
	return []Objective{
		{
			Name:        "detection_rate",
			Weight:      0.25,
			Target:      0.94,
			Type:        Maximize,
			Description: "Balanced threat detection",
			Unit:        "percentage",
		},
		{
			Name:        "false_positive_rate",
			Weight:      0.25,
			Target:      0.06,
			Type:        Minimize,
			Description: "Balanced false positive rate",
			Unit:        "percentage",
		},
		{
			Name:        "response_time_ms",
			Weight:      0.25,
			Target:      15.0,
			Type:        Minimize,
			Description: "Balanced response time",
			Unit:        "milliseconds",
		},
		{
			Name:        "user_satisfaction",
			Weight:      0.25,
			Target:      0.85,
			Type:        Maximize,
			Description: "Balanced user experience",
			Unit:        "score",
		},
	}
}

// NewObjectiveSet creates a validated objective set
func NewObjectiveSet(objectives []Objective) (*ObjectiveSet, error) {
	if len(objectives) == 0 {
		return nil, fmt.Errorf("objectives cannot be empty")
	}
	
	totalWeight := 0.0
	names := make(map[string]bool)
	
	for _, obj := range objectives {
		// Check for duplicate names
		if names[obj.Name] {
			return nil, fmt.Errorf("duplicate objective name: %s", obj.Name)
		}
		names[obj.Name] = true
		
		// Validate weight
		if obj.Weight < 0 || obj.Weight > 1 {
			return nil, fmt.Errorf("objective weight must be between 0 and 1: %s", obj.Name)
		}
		
		totalWeight += obj.Weight
		
		// Validate target values
		if err := validateObjectiveTarget(obj); err != nil {
			return nil, fmt.Errorf("invalid target for %s: %w", obj.Name, err)
		}
	}
	
	// Check if weights sum to approximately 1.0
	if totalWeight < 0.99 || totalWeight > 1.01 {
		return nil, fmt.Errorf("objective weights must sum to 1.0, got: %.3f", totalWeight)
	}
	
	return &ObjectiveSet{
		Objectives:  objectives,
		totalWeight: totalWeight,
	}, nil
}

// validateObjectiveTarget validates that objective targets are reasonable
func validateObjectiveTarget(obj Objective) error {
	switch obj.Name {
	case "detection_rate", "user_satisfaction":
		if obj.Target < 0 || obj.Target > 1 {
			return fmt.Errorf("target must be between 0 and 1")
		}
	case "false_positive_rate", "false_negative_rate":
		if obj.Target < 0 || obj.Target > 1 {
			return fmt.Errorf("rate must be between 0 and 1")
		}
	case "response_time_ms":
		if obj.Target < 0 || obj.Target > 1000 {
			return fmt.Errorf("response time must be between 0 and 1000ms")
		}
	case "throughput":
		if obj.Target < 0 {
			return fmt.Errorf("throughput must be positive")
		}
	}
	return nil
}

// CalculateScore calculates a weighted score based on actual metrics
func (os *ObjectiveSet) CalculateScore(metrics map[string]float64) float64 {
	score := 0.0
	
	for _, obj := range os.Objectives {
		value, exists := metrics[obj.Name]
		if !exists {
			continue // Skip missing metrics
		}
		
		// Normalize value (0-1 scale)
		normalizedValue := os.normalizeValue(obj, value)
		
		// Apply objective type
		if obj.Type == Minimize {
			// For minimize objectives, closer to 0 is better
			normalizedValue = 1.0 - normalizedValue
		}
		
		score += obj.Weight * normalizedValue
	}
	
	return score
}

// normalizeValue normalizes a metric value to 0-1 scale
func (os *ObjectiveSet) normalizeValue(obj Objective, value float64) float64 {
	switch obj.Name {
	case "detection_rate", "false_positive_rate", "false_negative_rate", "user_satisfaction":
		// Already normalized (0-1)
		return value
	case "response_time_ms":
		// Normalize based on target (target = 1.0, 0 = 0.0)
		if obj.Target == 0 {
			return 0
		}
		normalized := value / obj.Target
		if normalized > 1 {
			return 1 // Cap at 1.0
		}
		return normalized
	case "throughput":
		// Normalize based on target
		if obj.Target == 0 {
			return 0
		}
		normalized := value / obj.Target
		if normalized > 1 {
			return 1 // Cap at 1.0
		}
		return normalized
	default:
		// Default normalization for unknown metrics
		return value
	}
}

// GetObjectiveByName retrieves an objective by name
func (os *ObjectiveSet) GetObjectiveByName(name string) (*Objective, bool) {
	for _, obj := range os.Objectives {
		if obj.Name == name {
			return &obj, true
		}
	}
	return nil, false
}

// GetPrimaryObjective returns the objective with the highest weight
func (os *ObjectiveSet) GetPrimaryObjective() *Objective {
	if len(os.Objectives) == 0 {
		return nil
	}
	
	primary := &os.Objectives[0]
	for i := 1; i < len(os.Objectives); i++ {
		if os.Objectives[i].Weight > primary.Weight {
			primary = &os.Objectives[i]
		}
	}
	
	return primary
}

// ValidateMetrics checks if metrics meet minimum thresholds
func (os *ObjectiveSet) ValidateMetrics(metrics map[string]float64) []string {
	var failures []string
	
	for _, obj := range os.Objectives {
		value, exists := metrics[obj.Name]
		if !exists {
			failures = append(failures, fmt.Sprintf("missing metric: %s", obj.Name))
			continue
		}
		
		// Check if metric meets target
		if obj.Type == Maximize && value < obj.Target {
			failures = append(failures, 
				fmt.Sprintf("%s below target: %.3f < %.3f", obj.Name, value, obj.Target))
		} else if obj.Type == Minimize && value > obj.Target {
			failures = append(failures, 
				fmt.Sprintf("%s above target: %.3f > %.3f", obj.Name, value, obj.Target))
		}
	}
	
	return failures
}

// ObjectiveReport generates a detailed report of objective performance
type ObjectiveReport struct {
	TotalScore        float64                   `json:"total_score"`
	ObjectiveScores   map[string]float64        `json:"objective_scores"`
	MetricValues      map[string]float64        `json:"metric_values"`
	TargetComparison  map[string]ComparisonInfo `json:"target_comparison"`
	ValidationErrors  []string                  `json:"validation_errors"`
	GeneratedAt       time.Time                 `json:"generated_at"`
}

// ComparisonInfo contains comparison against target
type ComparisonInfo struct {
	Actual     float64 `json:"actual"`
	Target     float64 `json:"target"`
	Difference float64 `json:"difference"`
	MeetsTarget bool   `json:"meets_target"`
}

// GenerateReport creates a comprehensive objective performance report
func (os *ObjectiveSet) GenerateReport(metrics map[string]float64) *ObjectiveReport {
	report := &ObjectiveReport{
		ObjectiveScores:  make(map[string]float64),
		MetricValues:     make(map[string]float64),
		TargetComparison: make(map[string]ComparisonInfo),
		GeneratedAt:      time.Now(),
	}
	
	// Calculate total score
	report.TotalScore = os.CalculateScore(metrics)
	
	// Calculate individual objective scores
	for _, obj := range os.Objectives {
		value, exists := metrics[obj.Name]
		if !exists {
			continue
		}
		
		report.MetricValues[obj.Name] = value
		
		// Calculate individual score
		normalizedValue := os.normalizeValue(obj, value)
		if obj.Type == Minimize {
			normalizedValue = 1.0 - normalizedValue
		}
		report.ObjectiveScores[obj.Name] = obj.Weight * normalizedValue
		
		// Compare against target
		difference := value - obj.Target
		meetsTarget := (obj.Type == Maximize && value >= obj.Target) ||
					   (obj.Type == Minimize && value <= obj.Target)
		
		report.TargetComparison[obj.Name] = ComparisonInfo{
			Actual:      value,
			Target:      obj.Target,
			Difference:  difference,
			MeetsTarget: meetsTarget,
		}
	}
	
	// Validate metrics
	report.ValidationErrors = os.ValidateMetrics(metrics)
	
	return report
}

// Clone creates a deep copy of the objective set
func (os *ObjectiveSet) Clone() *ObjectiveSet {
	objectives := make([]Objective, len(os.Objectives))
	copy(objectives, os.Objectives)
	
	return &ObjectiveSet{
		Objectives:  objectives,
		totalWeight: os.totalWeight,
	}
}

// UpdateWeights updates objective weights with validation
func (os *ObjectiveSet) UpdateWeights(weights map[string]float64) error {
	// Create updated objectives
	updatedObjectives := make([]Objective, len(os.Objectives))
	copy(updatedObjectives, os.Objectives)
	
	totalWeight := 0.0
	for i, obj := range updatedObjectives {
		if newWeight, exists := weights[obj.Name]; exists {
			if newWeight < 0 || newWeight > 1 {
				return fmt.Errorf("weight for %s must be between 0 and 1", obj.Name)
			}
			updatedObjectives[i].Weight = newWeight
		}
		totalWeight += updatedObjectives[i].Weight
	}
	
	// Validate total weight
	if totalWeight < 0.99 || totalWeight > 1.01 {
		return fmt.Errorf("weights must sum to 1.0, got: %.3f", totalWeight)
	}
	
	// Update if valid
	os.Objectives = updatedObjectives
	os.totalWeight = totalWeight
	
	return nil
}