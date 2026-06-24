package analysis

import (
	"sort"
)

type Ranker struct{}

func NewRanker() *Ranker {
	return &Ranker{}
}

func (r *Ranker) Rank(hypotheses []Hypothesis, pattern *PatternMatch) {
	if pattern != nil {
		for i, h := range hypotheses {
			if h.Confidence > 0.9 {
				hypotheses[i].Rank = 1
				hypotheses[i].PatternHit = pattern
			}
		}
	}

	sort.Slice(hypotheses, func(i, j int) bool {
		if hypotheses[i].Rank != hypotheses[j].Rank {
			return hypotheses[i].Rank < hypotheses[j].Rank
		}
		return hypotheses[i].Confidence > hypotheses[j].Confidence
	})

	for i := range hypotheses {
		hypotheses[i].Rank = i + 1
	}
}
