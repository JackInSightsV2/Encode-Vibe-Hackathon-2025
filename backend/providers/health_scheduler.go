package providers

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// HealthScheduler manages periodic health checks
type HealthScheduler struct {
	monitor    *HealthMonitor
	interval   time.Duration
	ticker     *time.Ticker
	stopChan   chan struct{}
	running    bool
	mutex      sync.RWMutex
}

// NewHealthScheduler creates a new health check scheduler
func NewHealthScheduler(monitor *HealthMonitor, interval time.Duration) *HealthScheduler {
	return &HealthScheduler{
		monitor:  monitor,
		interval: interval,
		stopChan: make(chan struct{}),
		running:  false,
	}
}

// Start begins the health check schedule
func (hs *HealthScheduler) Start(ctx context.Context) error {
	hs.mutex.Lock()
	defer hs.mutex.Unlock()

	if hs.running {
		return nil // Already running
	}

	hs.ticker = time.NewTicker(hs.interval)
	hs.running = true

	// Start the scheduler goroutine
	go hs.run(ctx)

	return nil
}

// Stop stops the health check schedule
func (hs *HealthScheduler) Stop(ctx context.Context) error {
	hs.mutex.Lock()
	defer hs.mutex.Unlock()

	if !hs.running {
		return nil // Already stopped
	}

	// Signal stop
	close(hs.stopChan)

	// Stop ticker
	if hs.ticker != nil {
		hs.ticker.Stop()
	}

	hs.running = false

	return nil
}

// run is the main scheduler loop
func (hs *HealthScheduler) run(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Health scheduler panic recovered: %v\n", r)
		}
	}()

	// Perform initial health check
	hs.performHealthChecks(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-hs.stopChan:
			return
		case <-hs.ticker.C:
			hs.performHealthChecks(ctx)
		}
	}
}

// performHealthChecks runs health checks on all providers
func (hs *HealthScheduler) performHealthChecks(ctx context.Context) {
	// Get all providers from registry
	providers := hs.monitor.registry.List()

	// Create a context with timeout for all checks
	checkCtx, cancel := context.WithTimeout(ctx, hs.monitor.config.Timeout*time.Duration(len(providers)+1))
	defer cancel()

	// Use a wait group to perform checks concurrently
	var wg sync.WaitGroup
	
	for _, provider := range providers {
		wg.Add(1)
		go func(providerName string) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					fmt.Printf("Health check panic for provider %s: %v\n", providerName, r)
				}
			}()

			// Perform health check
			result := hs.monitor.CheckProvider(checkCtx, providerName)
			
			// Log health check results if needed
			if !result.Success {
				fmt.Printf("Health check failed for provider %s: %s\n", providerName, result.Error)
			}
		}(provider.Name())
	}

	// Wait for all checks to complete or timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-checkCtx.Done():
		fmt.Printf("Health checks timed out\n")
	case <-done:
		// All checks completed successfully
	}
}

// IsRunning returns whether the scheduler is currently running
func (hs *HealthScheduler) IsRunning() bool {
	hs.mutex.RLock()
	defer hs.mutex.RUnlock()
	return hs.running
}

// UpdateInterval updates the check interval (requires restart)
func (hs *HealthScheduler) UpdateInterval(interval time.Duration) {
	hs.mutex.Lock()
	defer hs.mutex.Unlock()
	
	hs.interval = interval
	
	// If running, update the ticker
	if hs.running && hs.ticker != nil {
		hs.ticker.Stop()
		hs.ticker = time.NewTicker(interval)
	}
}