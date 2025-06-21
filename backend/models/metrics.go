package models

import (
	"fmt"
	"time"
)

// Metric represents a system metric entry
type Metric struct {
	ID          int       `json:"id" db:"id"`
	MetricType  string    `json:"metric_type" db:"metric_type" validate:"required,max=50"`
	Name        string    `json:"name" db:"name" validate:"required,max=100"`
	Value       float64   `json:"value" db:"value" validate:"required"`
	Unit        *string   `json:"unit,omitempty" db:"unit" validate:"omitempty,max=20"`
	Tags        *string   `json:"tags,omitempty" db:"tags" validate:"omitempty,max=500"` // JSON string
	Labels      *string   `json:"labels,omitempty" db:"labels" validate:"omitempty,max=500"` // JSON string
	Source      string    `json:"source" db:"source" validate:"required,max=50"`
	Timestamp   time.Time `json:"timestamp" db:"timestamp"`
	CollectedAt time.Time `json:"collected_at" db:"collected_at"`
	Metadata    *string   `json:"metadata,omitempty" db:"metadata" validate:"omitempty,max=1000"` // JSON string
}

// MetricCreatePayload represents the payload for creating a metric
type MetricCreatePayload struct {
	MetricType string    `json:"metric_type" validate:"required,max=50"`
	Name       string    `json:"name" validate:"required,max=100"`
	Value      float64   `json:"value" validate:"required"`
	Unit       *string   `json:"unit,omitempty" validate:"omitempty,max=20"`
	Tags       *string   `json:"tags,omitempty" validate:"omitempty,max=500"`
	Labels     *string   `json:"labels,omitempty" validate:"omitempty,max=500"`
	Source     string    `json:"source" validate:"required,max=50"`
	Timestamp  time.Time `json:"timestamp,omitempty"`
	Metadata   *string   `json:"metadata,omitempty" validate:"omitempty,max=1000"`
}

// MetricFilters represents filters for querying metrics
type MetricFilters struct {
	MetricType  *string    `json:"metric_type,omitempty" validate:"omitempty,max=50"`
	Name        *string    `json:"name,omitempty" validate:"omitempty,max=100"`
	NamePattern *string    `json:"name_pattern,omitempty" validate:"omitempty,max=100"`
	Source      *string    `json:"source,omitempty" validate:"omitempty,max=50"`
	StartTime   *time.Time `json:"start_time,omitempty"`
	EndTime     *time.Time `json:"end_time,omitempty"`
	MinValue    *float64   `json:"min_value,omitempty"`
	MaxValue    *float64   `json:"max_value,omitempty"`
	Limit       int        `json:"limit" validate:"gte=1,lte=10000"`
	Offset      int        `json:"offset" validate:"gte=0"`
	OrderBy     string     `json:"order_by" validate:"omitempty,oneof=timestamp name value metric_type"`
	OrderDir    string     `json:"order_dir" validate:"omitempty,oneof=asc desc"`
	GroupBy     *string    `json:"group_by,omitempty" validate:"omitempty,oneof=name metric_type source hour day"`
	Aggregate   *string    `json:"aggregate,omitempty" validate:"omitempty,oneof=sum avg min max count"`
}

// MetricAggregation represents aggregated metric data
type MetricAggregation struct {
	Name        string    `json:"name"`
	MetricType  string    `json:"metric_type"`
	Source      string    `json:"source"`
	Value       float64   `json:"value"`
	Count       int       `json:"count"`
	Unit        *string   `json:"unit,omitempty"`
	Aggregation string    `json:"aggregation"` // sum, avg, min, max, count
	Period      string    `json:"period"`      // hour, day, etc.
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time"`
}

// MetricSummary represents a summary of metrics for a time period
type MetricSummary struct {
	TotalMetrics    int                        `json:"total_metrics"`
	MetricTypes     map[string]int             `json:"metric_types"`
	Sources         map[string]int             `json:"sources"`
	TopMetrics      []MetricRanking            `json:"top_metrics"`
	AverageValues   map[string]float64         `json:"average_values"`
	Period          string                     `json:"period"`
	StartTime       string                     `json:"start_time"`
	EndTime         string                     `json:"end_time"`
}

// MetricRanking represents a metric with its ranking information
type MetricRanking struct {
	Name       string  `json:"name"`
	MetricType string  `json:"metric_type"`
	Value      float64 `json:"value"`
	Count      int     `json:"count"`
	Source     string  `json:"source"`
}

// TimeSeries represents a time series data point
type TimeSeries struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
	Labels    map[string]string `json:"labels,omitempty"`
}

// MetricTimeSeriesResponse represents a response containing time series data
type MetricTimeSeriesResponse struct {
	Name       string       `json:"name"`
	MetricType string       `json:"metric_type"`
	Unit       *string      `json:"unit,omitempty"`
	Source     string       `json:"source"`
	Data       []TimeSeries `json:"data"`
	StartTime  time.Time    `json:"start_time"`
	EndTime    time.Time    `json:"end_time"`
	Interval   string       `json:"interval"`
}

// IsValid checks if the metric has valid data
func (m *Metric) IsValid() bool {
	return m.MetricType != "" && m.Name != "" && m.Source != ""
}

// IsCounterType checks if the metric is a counter type
func (m *Metric) IsCounterType() bool {
	return m.MetricType == "counter"
}

// IsGaugeType checks if the metric is a gauge type
func (m *Metric) IsGaugeType() bool {
	return m.MetricType == "gauge"
}

// IsHistogramType checks if the metric is a histogram type
func (m *Metric) IsHistogramType() bool {
	return m.MetricType == "histogram"
}

// GetDisplayValue returns the value formatted for display
func (m *Metric) GetDisplayValue() string {
	if m.Unit != nil {
		return fmt.Sprintf("%.2f %s", m.Value, *m.Unit)
	}
	return fmt.Sprintf("%.2f", m.Value)
}

// GetAge returns how old the metric is
func (m *Metric) GetAge() time.Duration {
	return time.Since(m.Timestamp)
}

// IsRecent checks if the metric was collected recently (within the last hour)
func (m *Metric) IsRecent() bool {
	return m.GetAge() < time.Hour
}