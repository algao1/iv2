package stats

import (
	"iv2/gourgeist/defs"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type StatsTestSuite struct {
	suite.Suite
}

func TestStatsTestSuite(t *testing.T) {
	suite.Run(t, new(StatsTestSuite))
}

func (suite *StatsTestSuite) TestTimeSpentInRange() {
	// trs := generateReadings(15, 60, 25, 4, 9)
	trs := genReadings([]metaReadings{
		{size: 15, min: 2, max: 4},
		{size: 60, min: 4, max: 9},
		{size: 25, min: 9, max: 20},
	}...)
	ra := TimeSpentInRange(trs, 4, 9)

	assert.Equal(suite.T(), 15.0/100, ra.BelowRange, "below range should match")
	assert.Equal(suite.T(), 60.0/100, ra.InRange, "in range should match")
	assert.Equal(suite.T(), 25.0/100, ra.AboveRange, "above range should match")
}

type metaReadings struct {
	size int
	min  float64
	max  float64
}

func genReadings(mrs ...metaReadings) []defs.TransformedReading {
	now := time.Now()
	trs := make([]defs.TransformedReading, 0)

	count := 0
	for _, mr := range mrs {
		for i := 0; i < mr.size; i++ {
			mmol := mr.min + rand.Float64()*(mr.max-mr.min)
			trs = append(trs, defs.TransformedReading{
				Time:  now.Add(time.Duration(count*5) * time.Minute),
				Mmol:  mmol,
				Trend: "Flat",
			})
			count++
		}
	}

	return trs
}
