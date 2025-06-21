package providers

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// ProviderCircuitBreaker implements circuit breaker pattern for providers
type ProviderCircuitBreaker struct {
	provider        string
	state           CircuitState
	failureCount    int
	successCount    int
	lastFailure     time.Time
	lastSuccess     time.Time
	config          *CircuitBreakerConfig
	mutex           sync.RWMutex
	stateChangeTime time.Time
}

// CircuitState represents the current state of the circuit breaker
type CircuitState string

const (
	StateClosed   CircuitState = "closed"   // Normal operation
	StateOpen     CircuitState = "open"     // Failing fast
	StateHalfOpen CircuitState = "half_open" // Testing recovery
)

// CircuitBreakerConfig contains circuit breaker configuration
type CircuitBreakerConfig struct {
	FailureThreshold    int           `yaml:"failure_threshold" json:"failure_threshold"`
	RecoveryTimeout     time.Duration `yaml:"recovery_timeout" json:"recovery_timeout"`
	HalfOpenMaxCalls    int           `yaml:"half_open_max_calls" json:"half_open_max_calls"`
	HalfOpenSuccessThreshold int      `yaml:"half_open_success_threshold" json:"half_open_success_threshold"`
	MinRequestsThreshold int          `yaml:"min_requests_threshold" json:"min_requests_threshold"`
}

// CircuitBreakerManager manages circuit breakers for all providers
type CircuitBreakerManager struct {
	breakers map[string]*ProviderCircuitBreaker
	config   *CircuitBreakerConfig
	mutex    sync.RWMutex
}

// NewProviderCircuitBreaker creates a new circuit breaker for a provider
func NewProviderCircuitBreaker(provider string, config *CircuitBreakerConfig) *ProviderCircuitBreaker {
	if config == nil {
		config = &CircuitBreakerConfig{
			FailureThreshold:         5,
			RecoveryTimeout:          30 * time.Second,
			HalfOpenMaxCalls:         3,
			HalfOpenSuccessThreshold: 2,
			MinRequestsThreshold:     5,
		}
	}

	return &ProviderCircuitBreaker{
		provider:        provider,
		state:           StateClosed,
		config:          config,
		stateChangeTime: time.Now(),
	}
}

// NewCircuitBreakerManager creates a new circuit breaker manager
func NewCircuitBreakerManager(config *CircuitBreakerConfig) *CircuitBreakerManager {
	return &CircuitBreakerManager{
		breakers: make(map[string]*ProviderCircuitBreaker),
		config:   config,
	}
}

// GetCircuitBreaker gets or creates a circuit breaker for a provider
func (cbm *CircuitBreakerManager) GetCircuitBreaker(provider string) *ProviderCircuitBreaker {
	cbm.mutex.Lock()
	defer cbm.mutex.Unlock()

	if breaker, exists := cbm.breakers[provider]; exists {
		return breaker
	}

	breaker := NewProviderCircuitBreaker(provider, cbm.config)
	cbm.breakers[provider] = breaker
	return breaker
}

// GetAllStates returns the state of all circuit breakers
func (cbm *CircuitBreakerManager) GetAllStates() map[string]CircuitBreakerState {
	cbm.mutex.RLock()
	defer cbm.mutex.RUnlock()

	states := make(map[string]CircuitBreakerState)
	for provider, breaker := range cbm.breakers {
		states[provider] = breaker.GetState()
	}

	return states
}

// CircuitBreakerState contains detailed circuit breaker state information
type CircuitBreakerState struct {
	State           CircuitState  `json:"state"`
	FailureCount    int           `json:"failure_count"`
	SuccessCount    int           `json:"success_count"`
	LastFailure     time.Time     `json:"last_failure"`
	LastSuccess     time.Time     `json:"last_success"`
	StateChangeTime time.Time     `json:"state_change_time"`
	NextRetryTime   time.Time     `json:"next_retry_time,omitempty"`
	IsAllowed       bool          `json:"is_allowed"`
}

// CanExecute determines if a request can be executed through this circuit breaker
func (cb *ProviderCircuitBreaker) CanExecute() bool {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	now := time.Now()

	switch cb.state {
	case StateClosed:
		return true

	case StateOpen:
		// Check if we should transition to half-open
		if now.Sub(cb.lastFailure) >= cb.config.RecoveryTimeout {
			cb.state = StateHalfOpen
			cb.stateChangeTime = now
			cb.successCount = 0
			cb.failureCount = 0
			return true
		}
		return false

	case StateHalfOpen:
		// Allow limited requests in half-open state
		totalCalls := cb.successCount + cb.failureCount
		return totalCalls < cb.config.HalfOpenMaxCalls

	default:
		return false
	}
}

// Execute executes a function with circuit breaker protection
func (cb *ProviderCircuitBreaker) Execute(ctx context.Context, fn func(ctx context.Context) error) error {
	if !cb.CanExecute() {
		return &CircuitBreakerError{
			Provider: cb.provider,
			State:    cb.state,
			Message:  "Circuit breaker is open",
		}
	}

	// Execute the function
	err := fn(ctx)

	// Record the result
	if err != nil {
		cb.RecordFailure()
	} else {
		cb.RecordSuccess()
	}

	return err
}

// RecordSuccess records a successful operation
func (cb *ProviderCircuitBreaker) RecordSuccess() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	cb.lastSuccess = time.Now()

	switch cb.state {
	case StateClosed:
		cb.failureCount = 0

	case StateHalfOpen:
		cb.successCount++
		if cb.successCount >= cb.config.HalfOpenSuccessThreshold {
			// Transition back to closed
			cb.state = StateClosed
			cb.stateChangeTime = time.Now()
			cb.failureCount = 0
			cb.successCount = 0
		}
	}
}

// RecordFailure records a failed operation
func (cb *ProviderCircuitBreaker) RecordFailure() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	cb.lastFailure = time.Now()

	switch cb.state {
	case StateClosed:
		cb.failureCount++
		if cb.failureCount >= cb.config.FailureThreshold {
			// Transition to open
			cb.state = StateOpen
			cb.stateChangeTime = time.Now()
		}

	case StateHalfOpen:
		cb.failureCount++
		// Any failure in half-open state transitions back to open
		cb.state = StateOpen
		cb.stateChangeTime = time.Now()
		cb.successCount = 0
		cb.failureCount = 0
	}
}

// GetState returns the current state of the circuit breaker
func (cb *ProviderCircuitBreaker) GetState() CircuitBreakerState {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()

	state := CircuitBreakerState{
		State:           cb.state,
		FailureCount:    cb.failureCount,
		SuccessCount:    cb.successCount,
		LastFailure:     cb.lastFailure,
		LastSuccess:     cb.lastSuccess,
		StateChangeTime: cb.stateChangeTime,
		IsAllowed:       cb.canExecuteInternal(),
	}

	// Calculate next retry time for open state
	if cb.state == StateOpen {
		state.NextRetryTime = cb.lastFailure.Add(cb.config.RecoveryTimeout)
	}

	return state
}

// canExecuteInternal is the internal version without mutex locking
func (cb *ProviderCircuitBreaker) canExecuteInternal() bool {
	now := time.Now()

	switch cb.state {
	case StateClosed:
		return true

	case StateOpen:
		return now.Sub(cb.lastFailure) >= cb.config.RecoveryTimeout

	case StateHalfOpen:
		totalCalls := cb.successCount + cb.failureCount
		return totalCalls < cb.config.HalfOpenMaxCalls

	default:
		return false
	}
}

// Reset resets the circuit breaker to closed state
func (cb *ProviderCircuitBreaker) Reset() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	cb.state = StateClosed
	cb.failureCount = 0
	cb.successCount = 0
	cb.stateChangeTime = time.Now()
}

// GetMetrics returns circuit breaker metrics
func (cb *ProviderCircuitBreaker) GetMetrics() map[string]interface{} {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()

	uptime := time.Since(cb.stateChangeTime)
	
	return map[string]interface{}{
		"provider":         cb.provider,
		"state":            string(cb.state),
		"failure_count":    cb.failureCount,
		"success_count":    cb.successCount,
		"last_failure":     cb.lastFailure.Format(time.RFC3339),
		"last_success":     cb.lastSuccess.Format(time.RFC3339),
		"state_uptime_ms":  uptime.Milliseconds(),
		"is_allowed":       cb.canExecuteInternal(),
		"config":           cb.config,
	}
}

// CircuitBreakerError represents an error when circuit breaker is open
type CircuitBreakerError struct {
	Provider string
	State    CircuitState
	Message  string
}

func (e *CircuitBreakerError) Error() string {
	return fmt.Sprintf("circuit breaker error for provider %s (state: %s): %s", 
		e.Provider, e.State, e.Message)
}

// IsCircuitBreakerError checks if an error is a circuit breaker error
func IsCircuitBreakerError(err error) bool {
	_, ok := err.(*CircuitBreakerError)
	return ok
}

// Enhanced Provider Wrapper with Circuit Breaker
type CircuitBreakerProvider struct {
	Provider
	circuitBreaker *ProviderCircuitBreaker
}

// NewCircuitBreakerProvider wraps a provider with circuit breaker functionality
func NewCircuitBreakerProvider(provider Provider, config *CircuitBreakerConfig) *CircuitBreakerProvider {
	return &CircuitBreakerProvider{
		Provider:       provider,
		circuitBreaker: NewProviderCircuitBreaker(provider.Name(), config),
	}
}

// SendRequest overrides the provider's SendRequest with circuit breaker protection
func (cbp *CircuitBreakerProvider) SendRequest(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	return cbp.executeWithCircuitBreaker(ctx, func(ctx context.Context) (*ChatResponse, error) {
		return cbp.Provider.SendRequest(ctx, req)
	})
}

// HealthCheck overrides the provider's HealthCheck with circuit breaker protection
func (cbp *CircuitBreakerProvider) HealthCheck(ctx context.Context) error {
	return cbp.circuitBreaker.Execute(ctx, func(ctx context.Context) error {
		return cbp.Provider.HealthCheck(ctx)
	})
}

// executeWithCircuitBreaker executes a function that returns a value with circuit breaker protection
func (cbp *CircuitBreakerProvider) executeWithCircuitBreaker(ctx context.Context, fn func(ctx context.Context) (*ChatResponse, error)) (*ChatResponse, error) {
	if !cbp.circuitBreaker.CanExecute() {
		return nil, &CircuitBreakerError{
			Provider: cbp.Provider.Name(),
			State:    cbp.circuitBreaker.state,
			Message:  "Circuit breaker is open",
		}
	}

	// Execute the function
	response, err := fn(ctx)

	// Record the result
	if err != nil {
		cbp.circuitBreaker.RecordFailure()
	} else {
		cbp.circuitBreaker.RecordSuccess()
	}

	return response, err
}

// GetCircuitBreakerState returns the circuit breaker state
func (cbp *CircuitBreakerProvider) GetCircuitBreakerState() CircuitBreakerState {
	return cbp.circuitBreaker.GetState()
}

// ResetCircuitBreaker resets the circuit breaker
func (cbp *CircuitBreakerProvider) ResetCircuitBreaker() {
	cbp.circuitBreaker.Reset()
}

// ProviderWithCircuitBreaker creates a provider wrapper with circuit breaker
func ProviderWithCircuitBreaker(provider Provider, config *CircuitBreakerConfig) Provider {
	return NewCircuitBreakerProvider(provider, config)
}