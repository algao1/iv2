package stats

import (
	"iv2/gourgeist/defs"
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
		case tr.Mmol <= 4:
			below++
		case tr.Mmol >= 9:
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
