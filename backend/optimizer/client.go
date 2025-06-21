package optimizer

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"qt1-middleware/opik"
)

// OptimizerClient manages optimization experiments and rule evolution
type OptimizerClient struct {
	opikClient  *opik.OpikClient
	projectID   string
	experiments map[string]*Experiment
	objectives  []Objective
	config      *OptimizerConfig
	mu          sync.RWMutex
	
	// Python engine communication
	pythonEngine *PythonEngine
}

// OptimizerConfig contains configuration for the optimizer
type OptimizerConfig struct {
	Enabled            bool          `yaml:"enabled" json:"enabled"`
	ProjectName        string        `yaml:"project_name" json:"project_name"`
	MaxWorkers         int           `yaml:"max_workers" json:"max_workers"`
	ExperimentDuration time.Duration `yaml:"experiment_duration" json:"experiment_duration"`
	MinSampleSize      int           `yaml:"min_sample_size" json:"min_sample_size"`
	ConfidenceLevel    float64       `yaml:"confidence_level" json:"confidence_level"`
	PythonScriptPath   string        `yaml:"python_script_path" json:"python_script_path"`
	
	// Optimization parameters
	PopulationSize   int     `yaml:"population_size" json:"population_size"`
	MutationRate     float64 `yaml:"mutation_rate" json:"mutation_rate"`
	CrossoverRate    float64 `yaml:"crossover_rate" json:"crossover_rate"`
	ElitismRate      float64 `yaml:"elitism_rate" json:"elitism_rate"`
	
	// Safety thresholds
	MinDetectionRate     float64 `yaml:"min_detection_rate" json:"min_detection_rate"`
	MaxFalsePositiveRate float64 `yaml:"max_false_positive_rate" json:"max_false_positive_rate"`
	MaxResponseTime      int     `yaml:"max_response_time_ms" json:"max_response_time_ms"`
}

// NewOptimizerClient creates a new optimizer client
func NewOptimizerClient(opikClient *opik.OpikClient, config *OptimizerConfig) (*OptimizerClient, error) {
	if config == nil {
		config = GetDefaultOptimizerConfig()
	}
	
	client := &OptimizerClient{
		opikClient:  opikClient,
		projectID:   config.ProjectName,
		experiments: make(map[string]*Experiment),
		objectives:  GetDefaultObjectives(),
		config:      config,
	}
	
	// Initialize Python engine
	pythonEngine, err := NewPythonEngine(config.PythonScriptPath)
	if err != nil {
		log.Printf("Warning: Python engine initialization failed: %v", err)
		// Continue without Python engine for basic functionality
	} else {
		client.pythonEngine = pythonEngine
	}
	
	return client, nil
}

// GetDefaultOptimizerConfig returns default configuration
func GetDefaultOptimizerConfig() *OptimizerConfig {
	return &OptimizerConfig{
		Enabled:            true,
		ProjectName:        "qt1-rule-optimizer",
		MaxWorkers:         10,
		ExperimentDuration: 1 * time.Hour,
		MinSampleSize:      1000,
		ConfidenceLevel:    0.95,
		PythonScriptPath:   "./optimizer/python/",
		
		PopulationSize:  100,
		MutationRate:    0.1,
		CrossoverRate:   0.3,
		ElitismRate:     0.1,
		
		MinDetectionRate:     0.95,
		MaxFalsePositiveRate: 0.05,
		MaxResponseTime:      10,
	}
}

// CreateExperiment creates a new optimization experiment
func (oc *OptimizerClient) CreateExperiment(config ExperimentConfig) (*Experiment, error) {
	oc.mu.Lock()
	defer oc.mu.Unlock()
	
	experiment := &Experiment{
		ID:          generateExperimentID(),
		Name:        config.Name,
		Status:      ExperimentStatusPending,
		StartTime:   time.Now(),
		Objectives:  config.Objectives,
		Control:     config.Control,
		Variants:    config.Variants,
		Traffic:     config.Traffic,
		Config:      config,
		Results:     NewExperimentResults(),
		opikClient:  oc.opikClient,
	}
	
	// Create Opik trace for experiment tracking
	if oc.opikClient != nil && oc.config.Enabled {
		ctx := context.Background()
		traceOptions := opik.TraceOptions{
			Input: map[string]interface{}{
				"experiment_name": experiment.Name,
				"control_rule":    experiment.Control,
				"variants_count":  len(experiment.Variants),
			},
			Metadata: map[string]interface{}{
				"experiment_type": "rule_optimization",
				"objectives":      experiment.Objectives,
			},
			Tags: []string{"optimization", "rules", "a-b-test"},
		}
		
		trace, err := oc.opikClient.StartTrace(ctx, fmt.Sprintf("rule_optimization_%s", experiment.Name), traceOptions)
		if err != nil {
			log.Printf("Failed to create Opik trace: %v", err)
		} else {
			experiment.OpikID = trace.ID
		}
	}
	
	oc.experiments[experiment.ID] = experiment
	
	log.Printf("Created experiment: %s (%s)", experiment.Name, experiment.ID)
	return experiment, nil
}

// GetExperiment retrieves an experiment by ID
func (oc *OptimizerClient) GetExperiment(id string) (*Experiment, bool) {
	oc.mu.RLock()
	defer oc.mu.RUnlock()
	
	experiment, exists := oc.experiments[id]
	return experiment, exists
}

// ListExperiments returns all experiments
func (oc *OptimizerClient) ListExperiments() []*Experiment {
	oc.mu.RLock()
	defer oc.mu.RUnlock()
	
	experiments := make([]*Experiment, 0, len(oc.experiments))
	for _, exp := range oc.experiments {
		experiments = append(experiments, exp)
	}
	
	return experiments
}

// GetActiveExperiments returns currently running experiments
func (oc *OptimizerClient) GetActiveExperiments() []*Experiment {
	oc.mu.RLock()
	defer oc.mu.RUnlock()
	
	var active []*Experiment
	for _, exp := range oc.experiments {
		if exp.Status == ExperimentStatusRunning {
			active = append(active, exp)
		}
	}
	
	return active
}

// StartExperiment begins running an experiment
func (oc *OptimizerClient) StartExperiment(experimentID string) error {
	oc.mu.Lock()
	defer oc.mu.Unlock()
	
	experiment, exists := oc.experiments[experimentID]
	if !exists {
		return fmt.Errorf("experiment not found: %s", experimentID)
	}
	
	if experiment.Status != ExperimentStatusPending {
		return fmt.Errorf("experiment not in pending state: %s", experiment.Status)
	}
	
	experiment.Status = ExperimentStatusRunning
	experiment.StartTime = time.Now()
	
	log.Printf("Started experiment: %s", experiment.Name)
	
	// Schedule automatic completion
	go oc.scheduleExperimentCompletion(experimentID)
	
	return nil
}

// StopExperiment stops a running experiment
func (oc *OptimizerClient) StopExperiment(experimentID, reason string) error {
	oc.mu.Lock()
	defer oc.mu.Unlock()
	
	experiment, exists := oc.experiments[experimentID]
	if !exists {
		return fmt.Errorf("experiment not found: %s", experimentID)
	}
	
	if experiment.Status != ExperimentStatusRunning {
		return fmt.Errorf("experiment not running: %s", experiment.Status)
	}
	
	experiment.Status = ExperimentStatusCompleted
	experiment.EndTime = &time.Time{}
	*experiment.EndTime = time.Now()
	experiment.StopReason = reason
	
	// Analyze results
	if err := oc.analyzeExperimentResults(experiment); err != nil {
		log.Printf("Failed to analyze experiment results: %v", err)
	}
	
	log.Printf("Stopped experiment: %s (reason: %s)", experiment.Name, reason)
	return nil
}

// GenerateRuleVariants creates new rule variants using the Python engine
func (oc *OptimizerClient) GenerateRuleVariants(baseRule Rule, count int) ([]Rule, error) {
	if oc.pythonEngine == nil {
		return oc.generateRuleVariantsBasic(baseRule, count)
	}
	
	// Use Python engine for advanced generation
	request := PythonRequest{
		Action: "generate_variants",
		Params: map[string]interface{}{
			"base_rule": baseRule,
			"count":     count,
			"objectives": oc.objectives,
		},
	}
	
	response, err := oc.pythonEngine.Execute(request)
	if err != nil {
		log.Printf("Python engine failed, falling back to basic generation: %v", err)
		return oc.generateRuleVariantsBasic(baseRule, count)
	}
	
	var variants []Rule
	if err := json.Unmarshal(response.Data, &variants); err != nil {
		return nil, fmt.Errorf("failed to parse variants: %w", err)
	}
	
	return variants, nil
}

// generateRuleVariantsBasic creates basic rule variants without ML
func (oc *OptimizerClient) generateRuleVariantsBasic(baseRule Rule, count int) ([]Rule, error) {
	variants := make([]Rule, 0, count)
	
	for i := 0; i < count; i++ {
		variant := baseRule.Clone()
		variant.ID = generateRuleID()
		variant.Name = fmt.Sprintf("%s_variant_%d", baseRule.Name, i+1)
		
		// Apply basic mutations
		if err := oc.applyBasicMutation(&variant); err != nil {
			log.Printf("Failed to apply mutation: %v", err)
			continue
		}
		
		variants = append(variants, variant)
	}
	
	return variants, nil
}

// applyBasicMutation applies simple parameter mutations
func (oc *OptimizerClient) applyBasicMutation(rule *Rule) error {
	// This is a simplified mutation - in practice, this would be more sophisticated
	switch rule.Type {
	case RuleTypeRegex:
		// Mutate regex parameters
		if params, ok := rule.Parameters["pattern"].(string); ok {
			// Simple pattern variation (this is very basic)
			rule.Parameters["pattern"] = params + "?"
		}
	case RuleTypeSemantic:
		// Mutate semantic thresholds
		if threshold, ok := rule.Parameters["threshold"].(float64); ok {
			variation := (threshold * 0.1) // 10% variation
			rule.Parameters["threshold"] = threshold + variation
		}
	}
	
	return nil
}

// scheduleExperimentCompletion schedules automatic experiment completion
func (oc *OptimizerClient) scheduleExperimentCompletion(experimentID string) {
	time.Sleep(oc.config.ExperimentDuration)
	
	if err := oc.StopExperiment(experimentID, "duration_completed"); err != nil {
		log.Printf("Failed to auto-complete experiment %s: %v", experimentID, err)
	}
}

// analyzeExperimentResults analyzes completed experiment results
func (oc *OptimizerClient) analyzeExperimentResults(experiment *Experiment) error {
	// Calculate final metrics for each variant
	for variantID := range experiment.Traffic.Variants {
		metrics := experiment.Results.GetVariantMetrics(variantID)
		if metrics != nil {
			// Calculate overall fitness score
			metrics.FitnessScore = oc.calculateFitnessScore(metrics)
		}
	}
	
	// Determine winner
	winner := experiment.Results.GetBestVariant()
	if winner != nil {
		experiment.Results.Winner = winner
		log.Printf("Experiment %s winner: %s (fitness: %.3f)", 
			experiment.Name, winner.VariantID, winner.FitnessScore)
	}
	
	// Log to Opik
	if oc.opikClient != nil && experiment.OpikID != "" {
		oc.logExperimentToOpik(experiment)
	}
	
	return nil
}

// calculateFitnessScore calculates overall fitness based on multiple objectives
func (oc *OptimizerClient) calculateFitnessScore(metrics *VariantMetrics) float64 {
	score := 0.0
	
	for _, objective := range oc.objectives {
		weight := objective.Weight
		var value float64
		
		switch objective.Name {
		case "detection_rate":
			value = metrics.DetectionRate
		case "false_positive_rate":
			value = metrics.FalsePositiveRate
		case "response_time_ms":
			// Normalize response time (target is 10ms, max acceptable is 50ms)
			responseMs := metrics.AvgResponseTime.Seconds() * 1000
			if responseMs <= objective.Target {
				value = 1.0
			} else {
				// Scale down from 1.0 to 0.0 as response time increases
				value = 1.0 - (responseMs-objective.Target)/(50.0-objective.Target)
				if value < 0 {
					value = 0
				}
			}
		case "user_satisfaction":
			value = metrics.UserSatisfaction
		}
		
		// Apply objective type (maximize/minimize)
		if objective.Type == Minimize {
			value = 1.0 - value
		}
		
		score += weight * value
	}
	
	return score
}

// logExperimentToOpik logs experiment results to Opik
func (oc *OptimizerClient) logExperimentToOpik(experiment *Experiment) {
	if experiment.OpikID == "" || experiment.OpikID == "disabled" {
		return
	}
	
	output := map[string]interface{}{
		"experiment_id":     experiment.ID,
		"status":           experiment.Status,
		"duration_seconds": experiment.GetDuration().Seconds(),
		"variants_count":   len(experiment.Variants),
		"stop_reason":      experiment.StopReason,
		"results_summary":  experiment.Results.Summary,
		"winner":           experiment.Results.Winner,
		"objectives_met":   len(experiment.Results.ValidationErrors) == 0,
	}
	
	// Create a minimal trace to end (since we can't retrieve the original trace object)
	// In a real implementation, we would store the trace object in the experiment
	trace := &opik.Trace{
		ID:        experiment.OpikID,
		ProjectID: oc.projectID,
		Status:    "completed",
	}
	
	if err := oc.opikClient.EndTrace(trace, output); err != nil {
		log.Printf("Failed to log experiment to Opik: %v", err)
	}
}

// Close shuts down the optimizer client
func (oc *OptimizerClient) Close() error {
	oc.mu.Lock()
	defer oc.mu.Unlock()
	
	// Stop all running experiments
	for _, experiment := range oc.experiments {
		if experiment.Status == ExperimentStatusRunning {
			experiment.Status = ExperimentStatusStopped
			experiment.StopReason = "system_shutdown"
		}
	}
	
	// Close Python engine
	if oc.pythonEngine != nil {
		oc.pythonEngine.Close()
	}
	
	return nil
}

// Helper functions

func generateExperimentID() string {
	return fmt.Sprintf("exp_%d_%s", time.Now().Unix(), generateRandomString(8))
}

func generateRuleID() string {
	return fmt.Sprintf("rule_%d_%s", time.Now().Unix(), generateRandomString(6))
}

func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}