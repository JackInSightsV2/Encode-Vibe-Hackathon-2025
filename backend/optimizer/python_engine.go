package optimizer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

// PythonEngine manages communication with Python optimization scripts
type PythonEngine struct {
	scriptPath    string
	pythonCmd     string
	timeout       time.Duration
	mu            sync.Mutex
	isInitialized bool
}

// PythonRequest represents a request to the Python engine
type PythonRequest struct {
	Action string                 `json:"action"`
	Params map[string]interface{} `json:"params"`
}

// PythonResponse represents a response from the Python engine
type PythonResponse struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data,omitempty"`
	Error   string          `json:"error,omitempty"`
	Meta    map[string]interface{} `json:"meta,omitempty"`
}

// NewPythonEngine creates a new Python engine
func NewPythonEngine(scriptPath string) (*PythonEngine, error) {
	engine := &PythonEngine{
		scriptPath: scriptPath,
		pythonCmd:  "python3",
		timeout:    30 * time.Second,
	}
	
	// Try to initialize Python environment
	if err := engine.initialize(); err != nil {
		return nil, fmt.Errorf("failed to initialize Python engine: %w", err)
	}
	
	return engine, nil
}

// initialize sets up the Python environment and validates dependencies
func (pe *PythonEngine) initialize() error {
	pe.mu.Lock()
	defer pe.mu.Unlock()
	
	// Check if Python is available
	cmd := exec.Command(pe.pythonCmd, "--version")
	if err := cmd.Run(); err != nil {
		// Try python instead of python3
		pe.pythonCmd = "python"
		cmd = exec.Command(pe.pythonCmd, "--version")
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("Python not found")
		}
	}
	
	// Check required packages
	if err := pe.checkPythonPackages(); err != nil {
		log.Printf("Warning: Some Python packages missing: %v", err)
		// Continue without packages for basic functionality
	}
	
	// Create optimizer script if it doesn't exist
	if err := pe.ensureOptimizerScript(); err != nil {
		return fmt.Errorf("failed to create optimizer script: %w", err)
	}
	
	pe.isInitialized = true
	return nil
}

// checkPythonPackages verifies required Python packages are installed
func (pe *PythonEngine) checkPythonPackages() error {
	packages := []string{"numpy", "scikit-learn", "scipy"}
	
	for _, pkg := range packages {
		cmd := exec.Command(pe.pythonCmd, "-c", fmt.Sprintf("import %s", pkg))
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("package %s not found", pkg)
		}
	}
	
	return nil
}

// ensureOptimizerScript creates the Python optimizer script
func (pe *PythonEngine) ensureOptimizerScript() error {
	scriptFile := filepath.Join(pe.scriptPath, "optimizer.py")
	
	// Create the Python script content
	scriptContent := `#!/usr/bin/env python3
"""
QT-1 Rule Optimizer
Advanced optimization engine for moderation rules
"""

import json
import sys
import random
import math
from typing import Dict, List, Any, Optional
import argparse

# Try to import advanced packages
try:
    import numpy as np
    HAS_NUMPY = True
except ImportError:
    HAS_NUMPY = False

try:
    from sklearn.ensemble import RandomForestClassifier
    from sklearn.model_selection import cross_val_score
    HAS_SKLEARN = True
except ImportError:
    HAS_SKLEARN = False

class RuleOptimizer:
    """Advanced rule optimization using machine learning techniques"""
    
    def __init__(self):
        self.population_size = 100
        self.mutation_rate = 0.1
        self.crossover_rate = 0.3
        self.elitism_rate = 0.1
        
    def generate_variants(self, base_rule: Dict[str, Any], count: int, objectives: List[Dict]) -> List[Dict]:
        """Generate rule variants using genetic algorithms"""
        variants = []
        
        for i in range(count):
            variant = self._mutate_rule(base_rule.copy(), objectives)
            variant['id'] = f"{base_rule.get('id', 'rule')}_{i+1}"
            variant['name'] = f"{base_rule.get('name', 'rule')}_variant_{i+1}"
            variants.append(variant)
            
        return variants
    
    def _mutate_rule(self, rule: Dict[str, Any], objectives: List[Dict]) -> Dict[str, Any]:
        """Apply mutations to a rule based on objectives"""
        rule_type = rule.get('type', 'regex')
        parameters = rule.get('parameters', {})
        
        if rule_type == 'regex':
            parameters = self._mutate_regex_params(parameters)
        elif rule_type == 'semantic':
            parameters = self._mutate_semantic_params(parameters)
        elif rule_type == 'pii':
            parameters = self._mutate_pii_params(parameters)
            
        rule['parameters'] = parameters
        return rule
    
    def _mutate_regex_params(self, params: Dict[str, Any]) -> Dict[str, Any]:
        """Mutate regex rule parameters"""
        if 'pattern' in params and random.random() < self.mutation_rate:
            pattern = params['pattern']
            # Simple pattern mutations
            if '.*' not in pattern:
                params['pattern'] = pattern + '?'  # Make optional
        
        if 'flags' in params and random.random() < self.mutation_rate:
            flags = params.get('flags', '')
            if 'i' not in flags:
                params['flags'] = flags + 'i'  # Add case insensitive
                
        return params
    
    def _mutate_semantic_params(self, params: Dict[str, Any]) -> Dict[str, Any]:
        """Mutate semantic rule parameters"""
        if 'threshold' in params and random.random() < self.mutation_rate:
            threshold = params['threshold']
            # Add small random variation
            variation = random.uniform(-0.05, 0.05)
            new_threshold = max(0.0, min(1.0, threshold + variation))
            params['threshold'] = new_threshold
            
        if 'model' in params and random.random() < self.mutation_rate:
            models = ['bert', 'roberta', 'distilbert', 'gpt']
            current_model = params['model']
            available_models = [m for m in models if m != current_model]
            if available_models:
                params['model'] = random.choice(available_models)
                
        return params
    
    def _mutate_pii_params(self, params: Dict[str, Any]) -> Dict[str, Any]:
        """Mutate PII detection parameters"""
        if 'pii_types' in params and random.random() < self.mutation_rate:
            all_types = ['email', 'phone', 'ssn', 'credit_card', 'address', 'name']
            current_types = params.get('pii_types', [])
            
            # Add or remove a type
            if random.random() < 0.5 and len(current_types) < len(all_types):
                # Add a type
                available = [t for t in all_types if t not in current_types]
                if available:
                    current_types.append(random.choice(available))
            elif len(current_types) > 1:
                # Remove a type
                current_types.remove(random.choice(current_types))
                
            params['pii_types'] = current_types
            
        if 'sensitivity' in params and random.random() < self.mutation_rate:
            sensitivity = params['sensitivity']
            variation = random.uniform(-0.1, 0.1)
            new_sensitivity = max(0.0, min(1.0, sensitivity + variation))
            params['sensitivity'] = new_sensitivity
            
        return params
    
    def optimize_population(self, rules: List[Dict], metrics: List[Dict], objectives: List[Dict]) -> List[Dict]:
        """Optimize a population of rules using genetic algorithms"""
        if not HAS_NUMPY:
            return self._basic_optimization(rules, metrics, objectives)
            
        # Calculate fitness scores
        fitness_scores = []
        for i, rule in enumerate(rules):
            if i < len(metrics):
                score = self._calculate_fitness(metrics[i], objectives)
                fitness_scores.append(score)
            else:
                fitness_scores.append(0.0)
        
        # Select best rules (elitism)
        elite_count = max(1, int(len(rules) * self.elitism_rate))
        elite_indices = np.argsort(fitness_scores)[-elite_count:]
        elite_rules = [rules[i] for i in elite_indices]
        
        # Generate new population
        new_population = elite_rules.copy()
        
        while len(new_population) < len(rules):
            if random.random() < self.crossover_rate and len(elite_rules) >= 2:
                # Crossover
                parent1, parent2 = random.sample(elite_rules, 2)
                child = self._crossover(parent1, parent2)
            else:
                # Mutation
                parent = random.choice(elite_rules)
                child = self._mutate_rule(parent.copy(), objectives)
                
            new_population.append(child)
        
        return new_population[:len(rules)]
    
    def _basic_optimization(self, rules: List[Dict], metrics: List[Dict], objectives: List[Dict]) -> List[Dict]:
        """Basic optimization without numpy"""
        # Simple selection of best rules
        scored_rules = []
        for i, rule in enumerate(rules):
            if i < len(metrics):
                score = self._calculate_fitness(metrics[i], objectives)
                scored_rules.append((score, rule))
            else:
                scored_rules.append((0.0, rule))
        
        # Sort by score and select top half
        scored_rules.sort(key=lambda x: x[0], reverse=True)
        top_half = [rule for _, rule in scored_rules[:len(rules)//2]]
        
        # Generate variants from top performers
        new_rules = top_half.copy()
        while len(new_rules) < len(rules):
            base_rule = random.choice(top_half)
            variant = self._mutate_rule(base_rule.copy(), objectives)
            new_rules.append(variant)
        
        return new_rules
    
    def _calculate_fitness(self, metrics: Dict[str, float], objectives: List[Dict]) -> float:
        """Calculate fitness score based on objectives"""
        score = 0.0
        total_weight = 0.0
        
        for obj in objectives:
            name = obj['name']
            weight = obj['weight']
            obj_type = obj['type']
            
            if name in metrics:
                value = metrics[name]
                
                # Normalize value
                if obj_type == 'minimize':
                    # For minimize objectives, lower is better
                    normalized = 1.0 - min(1.0, value)
                else:
                    # For maximize objectives, higher is better
                    normalized = min(1.0, value)
                
                score += weight * normalized
                total_weight += weight
        
        return score / max(total_weight, 1.0)
    
    def _crossover(self, parent1: Dict, parent2: Dict) -> Dict:
        """Create child rule by combining parents"""
        child = parent1.copy()
        
        # Mix parameters
        params1 = parent1.get('parameters', {})
        params2 = parent2.get('parameters', {})
        
        child_params = {}
        for key in set(params1.keys()) | set(params2.keys()):
            if key in params1 and key in params2:
                # Choose randomly between parents
                child_params[key] = random.choice([params1[key], params2[key]])
            elif key in params1:
                child_params[key] = params1[key]
            else:
                child_params[key] = params2[key]
        
        child['parameters'] = child_params
        child['id'] = f"child_{random.randint(1000, 9999)}"
        
        return child

def main():
    """Main function for command line interface"""
    parser = argparse.ArgumentParser(description='QT-1 Rule Optimizer')
    parser.add_argument('--action', required=True, help='Action to perform')
    parser.add_argument('--input', required=True, help='Input JSON data')
    
    args = parser.parse_args()
    
    try:
        input_data = json.loads(args.input)
        optimizer = RuleOptimizer()
        
        if args.action == 'generate_variants':
            base_rule = input_data['params']['base_rule']
            count = input_data['params']['count']
            objectives = input_data['params']['objectives']
            
            variants = optimizer.generate_variants(base_rule, count, objectives)
            result = {'success': True, 'data': variants}
            
        elif args.action == 'optimize_population':
            rules = input_data['params']['rules']
            metrics = input_data['params']['metrics']
            objectives = input_data['params']['objectives']
            
            optimized = optimizer.optimize_population(rules, metrics, objectives)
            result = {'success': True, 'data': optimized}
            
        else:
            result = {'success': False, 'error': f'Unknown action: {args.action}'}
    
    except Exception as e:
        result = {'success': False, 'error': str(e)}
    
    print(json.dumps(result))

if __name__ == '__main__':
    main()
`

	// Write the script file
	return writeFile(scriptFile, scriptContent)
}

// Execute runs a Python optimization command
func (pe *PythonEngine) Execute(request PythonRequest) (*PythonResponse, error) {
	pe.mu.Lock()
	defer pe.mu.Unlock()
	
	if !pe.isInitialized {
		return nil, fmt.Errorf("Python engine not initialized")
	}
	
	// Prepare input
	inputData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	
	// Execute Python script
	scriptFile := filepath.Join(pe.scriptPath, "optimizer.py")
	cmd := exec.Command(pe.pythonCmd, scriptFile, "--action", request.Action, "--input", string(inputData))
	
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	
	// Set timeout
	done := make(chan error, 1)
	go func() {
		done <- cmd.Run()
	}()
	
	select {
	case err := <-done:
		if err != nil {
			return &PythonResponse{
				Success: false,
				Error:   fmt.Sprintf("Python execution failed: %v\nStderr: %s", err, stderr.String()),
			}, nil
		}
	case <-time.After(pe.timeout):
		cmd.Process.Kill()
		return &PythonResponse{
			Success: false,
			Error:   "Python execution timeout",
		}, nil
	}
	
	// Parse response
	var response PythonResponse
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		return &PythonResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to parse Python response: %v\nOutput: %s", err, stdout.String()),
		}, nil
	}
	
	return &response, nil
}

// GenerateRuleVariants generates rule variants using Python ML algorithms
func (pe *PythonEngine) GenerateRuleVariants(baseRule Rule, count int, objectives []Objective) ([]Rule, error) {
	request := PythonRequest{
		Action: "generate_variants",
		Params: map[string]interface{}{
			"base_rule":  baseRule,
			"count":      count,
			"objectives": objectives,
		},
	}
	
	response, err := pe.Execute(request)
	if err != nil {
		return nil, err
	}
	
	if !response.Success {
		return nil, fmt.Errorf("Python engine error: %s", response.Error)
	}
	
	var variants []Rule
	if err := json.Unmarshal(response.Data, &variants); err != nil {
		return nil, fmt.Errorf("failed to parse variants: %w", err)
	}
	
	return variants, nil
}

// OptimizePopulation optimizes a population of rules using genetic algorithms
func (pe *PythonEngine) OptimizePopulation(rules []Rule, metrics []map[string]float64, objectives []Objective) ([]Rule, error) {
	request := PythonRequest{
		Action: "optimize_population",
		Params: map[string]interface{}{
			"rules":      rules,
			"metrics":    metrics,
			"objectives": objectives,
		},
	}
	
	response, err := pe.Execute(request)
	if err != nil {
		return nil, err
	}
	
	if !response.Success {
		return nil, fmt.Errorf("Python engine error: %s", response.Error)
	}
	
	var optimizedRules []Rule
	if err := json.Unmarshal(response.Data, &optimizedRules); err != nil {
		return nil, fmt.Errorf("failed to parse optimized rules: %w", err)
	}
	
	return optimizedRules, nil
}

// SetTimeout sets the execution timeout for Python scripts
func (pe *PythonEngine) SetTimeout(timeout time.Duration) {
	pe.mu.Lock()
	defer pe.mu.Unlock()
	pe.timeout = timeout
}

// Close shuts down the Python engine
func (pe *PythonEngine) Close() error {
	pe.mu.Lock()
	defer pe.mu.Unlock()
	
	pe.isInitialized = false
	return nil
}

// writeFile writes content to a file, creating directories if needed
func writeFile(filename, content string) error {
	// This is a simplified file write - in practice you'd use proper file operations
	cmd := exec.Command("mkdir", "-p", filepath.Dir(filename))
	cmd.Run()
	
	cmd = exec.Command("tee", filename)
	cmd.Stdin = bytes.NewBufferString(content)
	return cmd.Run()
}