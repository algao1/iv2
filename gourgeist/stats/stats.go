package stats

import (
	"iv2/gourgeist/types"
)

type RangeAnalysis struct {
	BelowRange float64
	InRange    float64
	AboveRange float64
}

func TimeSpentInRange(trs []types.TransformedReading, lower, upper float64) RangeAnalysis {
	if len(trs) == 0 {
		return RangeAnalysis{}
	}

	below, in, above := 0.0, 0.0, 0.0
	for _, tr := range trs {
		if tr.Mmol <= lower {
			below++
		} else if tr.Mmol >= upper {
			above++
		} else {
			in++
		}
	}

	total := below + in + above
	return RangeAnalysis{
		BelowRange: below / total,
		InRange:    in / total,
		AboveRange: above / total,
	}
}
