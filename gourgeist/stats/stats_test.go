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
	trs := generateReadings(15, 60, 25, 4, 9)
	ra := TimeSpentInRange(trs, 4, 9)

	assert.Equal(suite.T(), 15.0/100, ra.BelowRange, "below range should match")
	assert.Equal(suite.T(), 60.0/100, ra.InRange, "in range should match")
	assert.Equal(suite.T(), 25.0/100, ra.AboveRange, "above range should match")
}

func generateReadings(below, in, above int, belowThreshold, aboveThreshold float64) []defs.TransformedReading {
	now := time.Now()
	n := below + in + above

	trs := make([]defs.TransformedReading, n)
	for i := 0; i < n; i++ {
		mmol := belowThreshold + rand.Float64()*(aboveThreshold-belowThreshold)
		if below > 0 {
			mmol = rand.Float64() * belowThreshold
			below--
		} else if above > 0 {
			mmol = aboveThreshold + rand.Float64()
			above--
		}
		trs[i] = defs.TransformedReading{
			Time:  now.Add(time.Duration(i*5) * time.Minute),
			Mmol:  mmol,
			Trend: "Flat",
		}
	}
	return trs
}
