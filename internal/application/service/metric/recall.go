package metric

import (
	"github.com/aiplusall/aiplusall-kb/internal/types"
)

// RecallMetric calculates recall for retrieval evaluation
type RecallMetric struct{}

// NewRecallMetric creates a new RecallMetric instance
func NewRecallMetric() *RecallMetric {
	return &RecallMetric{}
}

// Compute calculates the recall score
func (r *RecallMetric) Compute(metricInput *types.MetricInput) float64 {
	// Get ground truth and predicted IDs
	gts := metricInput.RetrievalGT
	ids := metricInput.RetrievalIDs

	// Average recall across all queries
	if len(gts) == 0 {
		return 0.0
	}

	sumRecall := 0.0
	for _, gt := range gts {
		if len(gt) == 0 {
			continue
		}
		gtSet := ToSet(gt)
		hits := Hit(ids, gtSet)
		sumRecall += float64(hits) / float64(len(gt))
	}

	return sumRecall / float64(len(gts))
}
