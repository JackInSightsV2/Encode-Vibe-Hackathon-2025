package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"qt1-middleware/models"
	"strings"
	"time"
)

// metricsRepository implements MetricsRepository interface
type metricsRepository struct {
	db *sql.DB
}

// NewMetricsRepository creates a new metrics repository
func NewMetricsRepository(db *sql.DB) MetricsRepository {
	return &metricsRepository{db: db}
}

// Create inserts a new metric entry
func (r *metricsRepository) Create(ctx context.Context, metric *models.Metric) error {
	query := `
		INSERT INTO metrics (metric_type, name, value, unit, tags, labels, source, timestamp, collected_at, metadata)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	
	if metric.Timestamp.IsZero() {
		metric.Timestamp = time.Now()
	}
	metric.CollectedAt = time.Now()
	
	result, err := r.db.ExecContext(ctx, query,
		metric.MetricType, metric.Name, metric.Value, metric.Unit,
		metric.Tags, metric.Labels, metric.Source, metric.Timestamp,
		metric.CollectedAt, metric.Metadata,
	)
	if err != nil {
		return fmt.Errorf("failed to create metric: %w", err)
	}
	
	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get created metric ID: %w", err)
	}
	
	metric.ID = int(id)
	return nil
}

// CreateBatch inserts multiple metrics in a single transaction
func (r *metricsRepository) CreateBatch(ctx context.Context, metrics []*models.Metric) error {
	if len(metrics) == 0 {
		return nil
	}
	
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()
	
	query := `
		INSERT INTO metrics (metric_type, name, value, unit, tags, labels, source, timestamp, collected_at, metadata)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	
	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()
	
	now := time.Now()
	
	for i, metric := range metrics {
		if metric.Timestamp.IsZero() {
			metric.Timestamp = now
		}
		metric.CollectedAt = now
		
		result, err := stmt.ExecContext(ctx,
			metric.MetricType, metric.Name, metric.Value, metric.Unit,
			metric.Tags, metric.Labels, metric.Source, metric.Timestamp,
			metric.CollectedAt, metric.Metadata,
		)
		if err != nil {
			return fmt.Errorf("failed to insert metric %d: %w", i, err)
		}
		
		id, err := result.LastInsertId()
		if err != nil {
			return fmt.Errorf("failed to get ID for metric %d: %w", i, err)
		}
		
		metric.ID = int(id)
	}
	
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit batch insert: %w", err)
	}
	
	return nil
}

// GetByID retrieves a metric by ID
func (r *metricsRepository) GetByID(ctx context.Context, id int) (*models.Metric, error) {
	query := `
		SELECT id, metric_type, name, value, unit, tags, labels, source, timestamp, collected_at, metadata
		FROM metrics WHERE id = ?
	`
	
	metric := &models.Metric{}
	var unit, tags, labels, metadata sql.NullString
	
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&metric.ID, &metric.MetricType, &metric.Name, &metric.Value,
		&unit, &tags, &labels, &metric.Source, &metric.Timestamp,
		&metric.CollectedAt, &metadata,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get metric by ID: %w", err)
	}
	
	if unit.Valid {
		metric.Unit = &unit.String
	}
	if tags.Valid {
		metric.Tags = &tags.String
	}
	if labels.Valid {
		metric.Labels = &labels.String
	}
	if metadata.Valid {
		metric.Metadata = &metadata.String
	}
	
	return metric, nil
}

// Delete deletes a metric by ID
func (r *metricsRepository) Delete(ctx context.Context, id int) error {
	query := `DELETE FROM metrics WHERE id = ?`
	
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete metric: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("metric not found")
	}
	
	return nil
}

// List retrieves metrics with filtering
func (r *metricsRepository) List(ctx context.Context, filters *models.MetricFilters) ([]*models.Metric, error) {
	baseQuery := `
		SELECT id, metric_type, name, value, unit, tags, labels, source, timestamp, collected_at, metadata
		FROM metrics
	`
	
	var conditions []string
	var args []interface{}
	
	// Apply filters
	if filters.MetricType != nil {
		conditions = append(conditions, "metric_type = ?")
		args = append(args, *filters.MetricType)
	}
	
	if filters.Name != nil {
		conditions = append(conditions, "name = ?")
		args = append(args, *filters.Name)
	}
	
	if filters.NamePattern != nil {
		conditions = append(conditions, "name LIKE ?")
		args = append(args, "%"+*filters.NamePattern+"%")
	}
	
	if filters.Source != nil {
		conditions = append(conditions, "source = ?")
		args = append(args, *filters.Source)
	}
	
	if filters.StartTime != nil {
		conditions = append(conditions, "timestamp >= ?")
		args = append(args, *filters.StartTime)
	}
	
	if filters.EndTime != nil {
		conditions = append(conditions, "timestamp <= ?")
		args = append(args, *filters.EndTime)
	}
	
	if filters.MinValue != nil {
		conditions = append(conditions, "value >= ?")
		args = append(args, *filters.MinValue)
	}
	
	if filters.MaxValue != nil {
		conditions = append(conditions, "value <= ?")
		args = append(args, *filters.MaxValue)
	}
	
	// Build query with conditions
	query := baseQuery
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	
	// Add ordering
	orderBy := "timestamp"
	if filters.OrderBy != "" {
		orderBy = filters.OrderBy
	}
	orderDir := "DESC"
	if filters.OrderDir == "asc" {
		orderDir = "ASC"
	}
	query += fmt.Sprintf(" ORDER BY %s %s", orderBy, orderDir)
	
	// Add pagination
	if filters.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", filters.Limit)
		if filters.Offset > 0 {
			query += fmt.Sprintf(" OFFSET %d", filters.Offset)
		}
	}
	
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list metrics: %w", err)
	}
	defer rows.Close()
	
	return r.scanMetrics(rows)
}

// GetByName retrieves metrics by name
func (r *metricsRepository) GetByName(ctx context.Context, name string, limit, offset int) ([]*models.Metric, error) {
	query := `
		SELECT id, metric_type, name, value, unit, tags, labels, source, timestamp, collected_at, metadata
		FROM metrics 
		WHERE name = ?
		ORDER BY timestamp DESC
		LIMIT ? OFFSET ?
	`
	
	rows, err := r.db.QueryContext(ctx, query, name, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get metrics by name: %w", err)
	}
	defer rows.Close()
	
	return r.scanMetrics(rows)
}

// GetByType retrieves metrics by type
func (r *metricsRepository) GetByType(ctx context.Context, metricType string, limit, offset int) ([]*models.Metric, error) {
	query := `
		SELECT id, metric_type, name, value, unit, tags, labels, source, timestamp, collected_at, metadata
		FROM metrics 
		WHERE metric_type = ?
		ORDER BY timestamp DESC
		LIMIT ? OFFSET ?
	`
	
	rows, err := r.db.QueryContext(ctx, query, metricType, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get metrics by type: %w", err)
	}
	defer rows.Close()
	
	return r.scanMetrics(rows)
}

// GetBySource retrieves metrics by source
func (r *metricsRepository) GetBySource(ctx context.Context, source string, limit, offset int) ([]*models.Metric, error) {
	query := `
		SELECT id, metric_type, name, value, unit, tags, labels, source, timestamp, collected_at, metadata
		FROM metrics 
		WHERE source = ?
		ORDER BY timestamp DESC
		LIMIT ? OFFSET ?
	`
	
	rows, err := r.db.QueryContext(ctx, query, source, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get metrics by source: %w", err)
	}
	defer rows.Close()
	
	return r.scanMetrics(rows)
}

// GetTimeSeries retrieves time series data for a metric
func (r *metricsRepository) GetTimeSeries(ctx context.Context, name string, startTime, endTime time.Time, interval string) (*models.MetricTimeSeriesResponse, error) {
	// For simplicity, we'll return raw data points for now
	// In a production system, you might want to aggregate by interval
	query := `
		SELECT timestamp, value, labels
		FROM metrics 
		WHERE name = ? AND timestamp BETWEEN ? AND ?
		ORDER BY timestamp ASC
	`
	
	rows, err := r.db.QueryContext(ctx, query, name, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get time series: %w", err)
	}
	defer rows.Close()
	
	var dataPoints []models.TimeSeries
	var metricType, source string
	var unitPtr *string
	
	for rows.Next() {
		var timestamp time.Time
		var value float64
		var labelsJSON sql.NullString
		
		if err := rows.Scan(&timestamp, &value, &labelsJSON); err != nil {
			return nil, fmt.Errorf("failed to scan time series row: %w", err)
		}
		
		dataPoint := models.TimeSeries{
			Timestamp: timestamp,
			Value:     value,
		}
		
		// TODO: Parse labels JSON if needed
		if labelsJSON.Valid {
			// Parse JSON labels here
		}
		
		dataPoints = append(dataPoints, dataPoint)
	}
	
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating time series rows: %w", err)
	}
	
	// Get metric metadata
	metaQuery := `
		SELECT DISTINCT metric_type, source, unit
		FROM metrics 
		WHERE name = ? 
		LIMIT 1
	`
	
	var unitSQL sql.NullString
	err = r.db.QueryRowContext(ctx, metaQuery, name).Scan(&metricType, &source, &unitSQL)
	if unitSQL.Valid {
		unitPtr = &unitSQL.String
	}
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to get metric metadata: %w", err)
	}
	
	return &models.MetricTimeSeriesResponse{
		Name:       name,
		MetricType: metricType,
		Unit:       unitPtr,
		Source:     source,
		Data:       dataPoints,
		StartTime:  startTime,
		EndTime:    endTime,
		Interval:   interval,
	}, nil
}

// DeleteOlderThan deletes metrics older than the specified time
func (r *metricsRepository) DeleteOlderThan(ctx context.Context, cutoff time.Time) (int, error) {
	query := `DELETE FROM metrics WHERE timestamp < ?`
	
	result, err := r.db.ExecContext(ctx, query, cutoff)
	if err != nil {
		return 0, fmt.Errorf("failed to delete old metrics: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	return int(rowsAffected), nil
}

// GetAggregated retrieves aggregated metric data
func (r *metricsRepository) GetAggregated(ctx context.Context, filters *models.MetricFilters) ([]*models.MetricAggregation, error) {
	// Build aggregation query based on filters
	var aggregateFunc string
	var aggregateType string
	
	if filters.Aggregate != nil {
		aggregateType = *filters.Aggregate
	}
	
	switch aggregateType {
	case "":
		aggregateFunc = "AVG"
		aggregateType = "avg"
	case "sum":
		aggregateFunc = "SUM"
	case "avg":
		aggregateFunc = "AVG"
	case "min":
		aggregateFunc = "MIN"
	case "max":
		aggregateFunc = "MAX"
	case "count":
		aggregateFunc = "COUNT"
	default:
		aggregateFunc = "AVG"
		aggregateType = "avg"
	}
	
	var groupByClause string
	var selectFields string
	var groupByType string
	
	if filters.GroupBy != nil {
		groupByType = *filters.GroupBy
	}
	
	switch groupByType {
	case "":
		selectFields = "name, metric_type, source"
		groupByClause = "GROUP BY name, metric_type, source"
	case "name":
		selectFields = "name, metric_type, source"
		groupByClause = "GROUP BY name, metric_type, source"
	case "metric_type":
		selectFields = "metric_type, metric_type as name, source"
		groupByClause = "GROUP BY metric_type, source"
	case "source":
		selectFields = "source, source as name, metric_type"
		groupByClause = "GROUP BY source, metric_type"
	default:
		selectFields = "name, metric_type, source"
		groupByClause = "GROUP BY name, metric_type, source"
	}
	
	query := fmt.Sprintf(`
		SELECT %s, %s(value) as value, COUNT(*) as count, unit
		FROM metrics
	`, selectFields, aggregateFunc)
	
	var conditions []string
	var args []interface{}
	
	// Apply filters (similar to List method)
	if filters.MetricType != nil {
		conditions = append(conditions, "metric_type = ?")
		args = append(args, *filters.MetricType)
	}
	
	if filters.Source != nil {
		conditions = append(conditions, "source = ?")
		args = append(args, *filters.Source)
	}
	
	if filters.StartTime != nil {
		conditions = append(conditions, "timestamp >= ?")
		args = append(args, *filters.StartTime)
	}
	
	if filters.EndTime != nil {
		conditions = append(conditions, "timestamp <= ?")
		args = append(args, *filters.EndTime)
	}
	
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	
	query += " " + groupByClause
	
	// Add ordering
	orderBy := "value"
	if filters.OrderBy != "" {
		orderBy = filters.OrderBy
	}
	orderDir := "DESC"
	if filters.OrderDir == "asc" {
		orderDir = "ASC"
	}
	query += fmt.Sprintf(" ORDER BY %s %s", orderBy, orderDir)
	
	// Add pagination
	if filters.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", filters.Limit)
		if filters.Offset > 0 {
			query += fmt.Sprintf(" OFFSET %d", filters.Offset)
		}
	}
	
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get aggregated metrics: %w", err)
	}
	defer rows.Close()
	
	var aggregations []*models.MetricAggregation
	
	for rows.Next() {
		agg := &models.MetricAggregation{}
		var unit sql.NullString
		
		err := rows.Scan(&agg.Name, &agg.MetricType, &agg.Source, &agg.Value, &agg.Count, &unit)
		if err != nil {
			return nil, fmt.Errorf("failed to scan aggregation: %w", err)
		}
		
		if unit.Valid {
			agg.Unit = &unit.String
		}
		
		agg.Aggregation = aggregateType
		agg.Period = groupByType
		
		if filters.StartTime != nil {
			agg.StartTime = *filters.StartTime
		}
		if filters.EndTime != nil {
			agg.EndTime = *filters.EndTime
		}
		
		aggregations = append(aggregations, agg)
	}
	
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating aggregation rows: %w", err)
	}
	
	return aggregations, nil
}

// GetSummary retrieves a summary of metrics for a time period
func (r *metricsRepository) GetSummary(ctx context.Context, startTime, endTime time.Time) (*models.MetricSummary, error) {
	// Get total metrics count
	totalQuery := `SELECT COUNT(*) FROM metrics WHERE timestamp BETWEEN ? AND ?`
	var totalMetrics int
	if err := r.db.QueryRowContext(ctx, totalQuery, startTime, endTime).Scan(&totalMetrics); err != nil {
		return nil, fmt.Errorf("failed to get total metrics: %w", err)
	}
	
	// Get metrics by type
	typeQuery := `
		SELECT metric_type, COUNT(*) 
		FROM metrics 
		WHERE timestamp BETWEEN ? AND ? 
		GROUP BY metric_type
	`
	rows, err := r.db.QueryContext(ctx, typeQuery, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get metrics by type: %w", err)
	}
	defer rows.Close()
	
	metricTypes := make(map[string]int)
	for rows.Next() {
		var metricType string
		var count int
		if err := rows.Scan(&metricType, &count); err != nil {
			return nil, fmt.Errorf("failed to scan metric type: %w", err)
		}
		metricTypes[metricType] = count
	}
	
	// Get metrics by source
	sourceQuery := `
		SELECT source, COUNT(*) 
		FROM metrics 
		WHERE timestamp BETWEEN ? AND ? 
		GROUP BY source
	`
	rows, err = r.db.QueryContext(ctx, sourceQuery, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get metrics by source: %w", err)
	}
	defer rows.Close()
	
	sources := make(map[string]int)
	for rows.Next() {
		var source string
		var count int
		if err := rows.Scan(&source, &count); err != nil {
			return nil, fmt.Errorf("failed to scan source: %w", err)
		}
		sources[source] = count
	}
	
	// Get top metrics
	topQuery := `
		SELECT name, metric_type, AVG(value) as avg_value, COUNT(*) as count, source
		FROM metrics 
		WHERE timestamp BETWEEN ? AND ? 
		GROUP BY name, metric_type, source
		ORDER BY count DESC
		LIMIT 10
	`
	rows, err = r.db.QueryContext(ctx, topQuery, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get top metrics: %w", err)
	}
	defer rows.Close()
	
	var topMetrics []models.MetricRanking
	for rows.Next() {
		var ranking models.MetricRanking
		if err := rows.Scan(&ranking.Name, &ranking.MetricType, &ranking.Value, &ranking.Count, &ranking.Source); err != nil {
			return nil, fmt.Errorf("failed to scan metric ranking: %w", err)
		}
		topMetrics = append(topMetrics, ranking)
	}
	
	return &models.MetricSummary{
		TotalMetrics: totalMetrics,
		MetricTypes:  metricTypes,
		Sources:      sources,
		TopMetrics:   topMetrics,
		Period:       fmt.Sprintf("%s to %s", startTime.Format(time.RFC3339), endTime.Format(time.RFC3339)),
		StartTime:    startTime.Format(time.RFC3339),
		EndTime:      endTime.Format(time.RFC3339),
	}, nil
}

// Count returns the total number of metrics
func (r *metricsRepository) Count(ctx context.Context) (int, error) {
	query := `SELECT COUNT(*) FROM metrics`
	
	var count int
	err := r.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count metrics: %w", err)
	}
	
	return count, nil
}

// CountByType returns the number of metrics by type
func (r *metricsRepository) CountByType(ctx context.Context, metricType string) (int, error) {
	query := `SELECT COUNT(*) FROM metrics WHERE metric_type = ?`
	
	var count int
	err := r.db.QueryRowContext(ctx, query, metricType).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count metrics by type: %w", err)
	}
	
	return count, nil
}

// CountBySource returns the number of metrics by source
func (r *metricsRepository) CountBySource(ctx context.Context, source string) (int, error) {
	query := `SELECT COUNT(*) FROM metrics WHERE source = ?`
	
	var count int
	err := r.db.QueryRowContext(ctx, query, source).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count metrics by source: %w", err)
	}
	
	return count, nil
}

// scanMetrics is a helper function to scan multiple metrics from rows
func (r *metricsRepository) scanMetrics(rows *sql.Rows) ([]*models.Metric, error) {
	var metrics []*models.Metric
	
	for rows.Next() {
		metric := &models.Metric{}
		var unit, tags, labels, metadata sql.NullString
		
		err := rows.Scan(
			&metric.ID, &metric.MetricType, &metric.Name, &metric.Value,
			&unit, &tags, &labels, &metric.Source, &metric.Timestamp,
			&metric.CollectedAt, &metadata,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan metric: %w", err)
		}
		
		if unit.Valid {
			metric.Unit = &unit.String
		}
		if tags.Valid {
			metric.Tags = &tags.String
		}
		if labels.Valid {
			metric.Labels = &labels.String
		}
		if metadata.Valid {
			metric.Metadata = &metadata.String
		}
		
		metrics = append(metrics, metric)
	}
	
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating metric rows: %w", err)
	}
	
	return metrics, nil
}