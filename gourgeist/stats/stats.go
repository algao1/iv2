package stats

import (
	"iv2/gourgeist/defs"

	"github.com/montanaflynn/stats"
)

type RangeAnalysis struct {
	BelowRange float64
	InRange    float64
	AboveRange float64
}

func TimeSpentInRange(trs []defs.TransformedReading, lower, upper float64) RangeAnalysis {
	if len(trs) == 0 {
		return RangeAnalysis{}
	}

	below, above := 0.0, 0.0
	for _, tr := range trs {
		switch {
		case tr.Mmol <= lower:
			below++
		case tr.Mmol >= upper:
			above++
		}
	}
	in := float64(len(trs)) - below - above

	total := float64(len(trs))
	return RangeAnalysis{
		BelowRange: below / total,
		InRange:    in / total,
		AboveRange: above / total,
	}
}

type SummaryStatistics struct {
	Average   float64
	Deviation float64
}

func GlucoseSummary(trs []defs.TransformedReading) SummaryStatistics {
	trFloats := make([]float64, len(trs))
	for i, tr := range trs {
		trFloats[i] = tr.Mmol
	}
	avg, _ := stats.Mean(trFloats)
	dev, _ := stats.StandardDeviation(trFloats)
	return SummaryStatistics{Average: avg, Deviation: dev}
}
