package metrics

import (
	"fmt"
	"log"
	"sync"
	"time"
)

// Aggregator handles background aggregation of time-series data
type Aggregator struct {
	storage   *TimeSeriesStorage
	config    TimeSeriesConfig
	stopChan  chan struct{}
	wg        sync.WaitGroup
	running   bool
	mutex     sync.RWMutex
}

// AggregationJob represents a single aggregation task
type AggregationJob struct {
	SourceResolution BucketResolution
	TargetResolution BucketResolution
	StartTime        time.Time
	EndTime          time.Time
}

// NewAggregator creates a new aggregator instance
func NewAggregator(storage *TimeSeriesStorage, config TimeSeriesConfig) *Aggregator {
	return &Aggregator{
		storage:  storage,
		config:   config,
		stopChan: make(chan struct{}),
		running:  false,
	}
}

// Start begins the aggregation engine
func (a *Aggregator) Start() {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	if a.running {
		return
	}

	a.running = true
	a.wg.Add(1)

	go a.aggregationLoop()
}

// Stop gracefully stops the aggregation engine
func (a *Aggregator) Stop() {
	a.mutex.RLock()
	running := a.running
	a.mutex.RUnlock()

	if !running {
		return
	}

	close(a.stopChan)
	a.wg.Wait()

	a.mutex.Lock()
	a.running = false
	a.mutex.Unlock()
}

// aggregationLoop runs the main aggregation process
func (a *Aggregator) aggregationLoop() {
	defer a.wg.Done()

	// Create ticker for aggregation interval
	interval := time.Duration(a.config.AggregationIntervalSecs) * time.Second
	if interval <= 0 {
		interval = 60 * time.Second // Default to 1 minute
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	log.Printf("Aggregator started with %v interval", interval)

	for {
		select {
		case <-ticker.C:
			a.performAggregation()
		case <-a.stopChan:
			log.Println("Aggregator stopping...")
			return
		}
	}
}

// performAggregation executes the aggregation process
func (a *Aggregator) performAggregation() {
	now := time.Now()
	
	// Define aggregation jobs in order (lower to higher resolution)
	jobs := []AggregationJob{
		{
			SourceResolution: Resolution1Min,
			TargetResolution: Resolution5Min,
			StartTime:        now.Add(-10 * time.Minute), // Process last 10 minutes
			EndTime:          now.Add(-5 * time.Minute),  // Leave 5-minute buffer
		},
		{
			SourceResolution: Resolution5Min,
			TargetResolution: Resolution1Hour,
			StartTime:        now.Add(-2 * time.Hour),
			EndTime:          now.Add(-1 * time.Hour),
		},
		{
			SourceResolution: Resolution1Hour,
			TargetResolution: Resolution1Day,
			StartTime:        now.Add(-48 * time.Hour),
			EndTime:          now.Add(-24 * time.Hour),
		},
	}

	for _, job := range jobs {
		if err := a.executeAggregationJob(job); err != nil {
			log.Printf("Aggregation job failed: %v", err)
		}
	}

	// Cleanup old buckets after aggregation
	if err := a.storage.cleanupTimeSeries(); err != nil {
		log.Printf("Cleanup failed: %v", err)
	}
}

// executeAggregationJob performs a single aggregation job
func (a *Aggregator) executeAggregationJob(job AggregationJob) error {
	a.storage.mutex.RLock()
	sourceBucketMap := a.storage.buckets[job.SourceResolution]
	targetBucketMap := a.storage.buckets[job.TargetResolution]
	a.storage.mutex.RUnlock()

	if sourceBucketMap == nil || targetBucketMap == nil {
		return fmt.Errorf("bucket map not found for resolution %s or %s", 
			job.SourceResolution, job.TargetResolution)
	}

	// Get source buckets in the time range
	sourceBuckets := a.storage.getBucketsInRange(sourceBucketMap, job.StartTime, job.EndTime)
	if len(sourceBuckets) == 0 {
		return nil // Nothing to aggregate
	}

	// Group source buckets by target bucket timestamp
	targetGroups := a.groupBucketsByTargetResolution(sourceBuckets, job.TargetResolution)

	// Aggregate each group into target buckets
	for targetTimestamp, sourceBucketsGroup := range targetGroups {
		if err := a.aggregateToTargetBucket(targetBucketMap, targetTimestamp, sourceBucketsGroup); err != nil {
			return fmt.Errorf("failed to aggregate to target bucket: %v", err)
		}
	}

	log.Printf("Aggregated %d buckets from %s to %s", 
		len(sourceBuckets), job.SourceResolution, job.TargetResolution)

	return nil
}

// groupBucketsByTargetResolution groups source buckets by their target resolution timestamp
func (a *Aggregator) groupBucketsByTargetResolution(sourceBuckets []*TimeBucket, targetResolution BucketResolution) map[time.Time][]*TimeBucket {
	groups := make(map[time.Time][]*TimeBucket)

	for _, bucket := range sourceBuckets {
		targetTimestamp := a.storage.truncateToResolution(bucket.Timestamp, targetResolution)
		groups[targetTimestamp] = append(groups[targetTimestamp], bucket)
	}

	return groups
}

// aggregateToTargetBucket aggregates multiple source buckets into a single target bucket
func (a *Aggregator) aggregateToTargetBucket(targetBucketMap *TimeBucketMap, targetTimestamp time.Time, sourceBuckets []*TimeBucket) error {
	targetBucketMap.mutex.Lock()
	defer targetBucketMap.mutex.Unlock()

	targetKey := targetTimestamp.Unix()
	
	// Get or create target bucket
	targetBucket, exists := targetBucketMap.buckets[targetKey]
	if !exists {
		targetBucket = &TimeBucket{
			Timestamp: targetTimestamp,
			Metrics:   make(map[string]*AggregatedValue),
			Count:     0,
		}
		targetBucketMap.buckets[targetKey] = targetBucket
	}

	// Aggregate metrics from all source buckets
	metricAggregates := make(map[string]*AggregatedValue)

	for _, sourceBucket := range sourceBuckets {
		for metricName, sourceValue := range sourceBucket.Metrics {
			if agg, exists := metricAggregates[metricName]; exists {
				// Update existing aggregate
				a.updateAggregate(agg, sourceValue)
			} else {
				// Create new aggregate
				metricAggregates[metricName] = &AggregatedValue{
					Sum:     sourceValue.Sum,
					Min:     sourceValue.Min,
					Max:     sourceValue.Max,
					Count:   sourceValue.Count,
					Average: sourceValue.Average,
					Latest:  sourceValue.Latest,
				}
			}
		}
		targetBucket.Count += sourceBucket.Count
	}

	// Update target bucket with aggregated values
	for metricName, aggregatedValue := range metricAggregates {
		if existing, exists := targetBucket.Metrics[metricName]; exists {
			// Merge with existing value in target bucket
			a.mergeAggregates(existing, aggregatedValue)
		} else {
			// Set new aggregated value
			targetBucket.Metrics[metricName] = aggregatedValue
		}
	}

	return nil
}

// updateAggregate updates an aggregate with a new source value
func (a *Aggregator) updateAggregate(target *AggregatedValue, source *AggregatedValue) {
	// Update sum
	target.Sum += source.Sum

	// Update min/max
	if source.Min < target.Min {
		target.Min = source.Min
	}
	if source.Max > target.Max {
		target.Max = source.Max
	}

	// Update count and recalculate average
	target.Count += source.Count
	if target.Count > 0 {
		target.Average = target.Sum / float64(target.Count)
	}

	// Update latest (use the most recent)
	target.Latest = source.Latest
}

// mergeAggregates merges two aggregated values
func (a *Aggregator) mergeAggregates(target *AggregatedValue, source *AggregatedValue) {
	// Weighted average calculation for merging
	totalCount := target.Count + source.Count
	if totalCount > 0 {
		target.Sum += source.Sum
		target.Average = target.Sum / float64(totalCount)
	}

	// Update min/max
	if source.Min < target.Min {
		target.Min = source.Min
	}
	if source.Max > target.Max {
		target.Max = source.Max
	}

	target.Count = totalCount
	target.Latest = source.Latest
}

// ForceAggregation manually triggers aggregation (useful for testing)
func (a *Aggregator) ForceAggregation() error {
	a.mutex.RLock()
	running := a.running
	a.mutex.RUnlock()

	if !running {
		return fmt.Errorf("aggregator is not running")
	}

	a.performAggregation()
	return nil
}

// GetStatus returns the current status of the aggregator
func (a *Aggregator) GetStatus() AggregatorStatus {
	a.mutex.RLock()
	defer a.mutex.RUnlock()

	return AggregatorStatus{
		Running:           a.running,
		AggregationInterval: time.Duration(a.config.AggregationIntervalSecs) * time.Second,
		BucketCounts:      a.getBucketCounts(),
	}
}

// AggregatorStatus represents the current status of the aggregator
type AggregatorStatus struct {
	Running             bool                       `json:"running"`
	AggregationInterval time.Duration              `json:"aggregation_interval"`
	BucketCounts        map[BucketResolution]int   `json:"bucket_counts"`
}

// getBucketCounts returns the number of buckets for each resolution
func (a *Aggregator) getBucketCounts() map[BucketResolution]int {
	counts := make(map[BucketResolution]int)

	a.storage.mutex.RLock()
	defer a.storage.mutex.RUnlock()

	for resolution, bucketMap := range a.storage.buckets {
		bucketMap.mutex.RLock()
		counts[resolution] = len(bucketMap.buckets)
		bucketMap.mutex.RUnlock()
	}

	return counts
}

// EstimateMemoryUsage provides an estimate of memory usage for time-series data
func (a *Aggregator) EstimateMemoryUsage() MemoryUsageEstimate {
	estimate := MemoryUsageEstimate{
		TotalBuckets: 0,
		EstimatedMB:  0,
	}

	bucketCounts := a.getBucketCounts()
	
	for resolution, count := range bucketCounts {
		estimate.TotalBuckets += count
		
		// Rough estimate: each bucket ~1KB, varies by resolution
		var bucketSizeKB float64
		switch resolution {
		case Resolution1Min:
			bucketSizeKB = 1.0  // High frequency, smaller buckets
		case Resolution5Min:
			bucketSizeKB = 2.0  // Medium frequency
		case Resolution1Hour:
			bucketSizeKB = 5.0  // Lower frequency, larger buckets
		case Resolution1Day:
			bucketSizeKB = 10.0 // Lowest frequency, largest buckets
		default:
			bucketSizeKB = 1.0
		}
		
		estimate.EstimatedMB += float64(count) * bucketSizeKB / 1024.0
	}

	estimate.BucketBreakdown = bucketCounts
	return estimate
}

// MemoryUsageEstimate provides memory usage information
type MemoryUsageEstimate struct {
	TotalBuckets     int                       `json:"total_buckets"`
	EstimatedMB      float64                   `json:"estimated_mb"`
	BucketBreakdown  map[BucketResolution]int  `json:"bucket_breakdown"`
}