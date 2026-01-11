package metric

import (
	"github.com/aiplusall/aiplusall-kb/internal/types"
)

// PrecisionMetric calculates precision for retrieval evaluation
type PrecisionMetric struct{}

// NewPrecisionMetric creates a new PrecisionMetric instance
func NewPrecisionMetric() *PrecisionMetric {
	return &PrecisionMetric{}
}

// Compute calculates the precision score
func (r *PrecisionMetric) Compute(metricInput *types.MetricInput) float64 {
	// Get ground truth and predicted IDs
	gts := metricInput.RetrievalGT
	ids := metricInput.RetrievalIDs

	// Average precision across all queries
	if len(gts) == 0 {
		return 0.0
	}

	sumPrecision := 0.0
	for _, gt := range gts {
		if len(ids) == 0 {
			continue
		}
		gtSet := ToSet(gt)
		hits := Hit(ids, gtSet)
		sumPrecision += float64(hits) / float64(len(ids))
	}

	return sumPrecision / float64(len(gts))
}
