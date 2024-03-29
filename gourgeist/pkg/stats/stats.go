package stats

import (
	"iv2/gourgeist/defs"
	"sort"
	"time"

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

// TODO: Add tests.

type IntakeData struct {
	Ins   []defs.Insulin
	Carbs []defs.Carb
}

type DailyData struct {
	Days     []time.Time // This will be in sorted order.
	InsMap   map[time.Time][]defs.Insulin
	CarbsMap map[time.Time][]defs.Carb
}

func DailyAggregate(data IntakeData, loc *time.Location) DailyData {
	im := make(map[time.Time][]defs.Insulin)
	cm := make(map[time.Time][]defs.Carb)
	keyMap := make(map[time.Time]struct{})

	for _, in := range data.Ins {
		toRound := in.Time.In(loc)
		rounded := time.Date(toRound.Year(), toRound.Month(), toRound.Day(), 0, 0, 0, 0, loc)
		if _, ok := im[rounded]; !ok {
			im[rounded] = make([]defs.Insulin, 0)
			keyMap[rounded] = struct{}{}
		}
		im[rounded] = append(im[rounded], in)
	}

	for _, c := range data.Carbs {
		toRound := c.Time.In(loc)
		rounded := time.Date(toRound.Year(), toRound.Month(), toRound.Day(), 0, 0, 0, 0, loc)
		if _, ok := cm[rounded]; !ok {
			cm[rounded] = make([]defs.Carb, 0)
			keyMap[rounded] = struct{}{}
		}
		cm[rounded] = append(cm[rounded], c)
	}

	keys := make([]time.Time, 0)
	for k := range keyMap {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		return keys[i].Before(keys[j])
	})

	return DailyData{Days: keys, InsMap: im, CarbsMap: cm}
}
