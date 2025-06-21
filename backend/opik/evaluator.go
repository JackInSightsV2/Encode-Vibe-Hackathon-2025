package opik

// Evaluator interface for custom evaluations
type Evaluator interface {
	Name() string
	Evaluate(trace *Trace) (float64, map[string]interface{})
}

// BaseEvaluator provides common functionality for evaluators
type BaseEvaluator struct {
	name      string
	threshold float64
}

// Name returns the evaluator name
func (e *BaseEvaluator) Name() string {
	return e.name
}

// NewBaseEvaluator creates a new base evaluator
func NewBaseEvaluator(name string, threshold float64) BaseEvaluator {
	return BaseEvaluator{
		name:      name,
		threshold: threshold,
	}
}