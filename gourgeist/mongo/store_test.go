package mongo

import (
	"context"
	"iv2/gourgeist/types"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
)

const (
	mongoURI = "mongodb://localhost:27017"
	testDB   = "test"
)

type MongoTestSuite struct {
	suite.Suite
	ms *MongoStore
}

func TestMongoTestSuiteIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	suite.Run(t, new(MongoTestSuite))
}

func (suite *MongoTestSuite) SetupSuite() {
	ms, err := New(context.Background(), mongoURI, testDB, zap.NewExample())
	if err != nil {
		panic(err)
	}
	suite.ms = ms
}

func (suite *MongoTestSuite) AfterTest(_, _ string) {
	suite.T().Log("teardown test db")
	assert.NoError(suite.T(), suite.ms.Client.Database(testDB).Drop(context.Background()), "unable to drop test db")
}

func (suite *MongoTestSuite) TestReadWriteGlucoseIntegration() {
	ctx := context.Background()
	times := []time.Time{
		time.Date(2022, time.May, 15, 1, 30, 0, 0, time.UTC), // Entry.
		time.Date(2022, time.May, 10, 0, 0, 0, 0, time.UTC),  // Start.
		time.Date(2022, time.May, 20, 0, 0, 0, 0, time.UTC),  // End.
	}
	tr := types.TransformedReading{
		Time:  times[0],
		Mmol:  6.5,
		Trend: "Flat",
	}

	replaced, err := suite.ms.WriteGlucose(ctx, &tr)
	assert.NoError(suite.T(), err, "unable to write glucose to test db")
	assert.False(suite.T(), replaced, "not unique entry")

	trs, err := suite.ms.ReadGlucose(ctx, times[1], times[2])
	assert.NoError(suite.T(), err, "unable to read glucose from test db")
	assert.Len(suite.T(), trs, 1, "did not find exactly one entry")
}

func (suite *MongoTestSuite) TestReadWriteInsulinIntegration() {
	ctx := context.Background()
	times := []time.Time{
		time.Date(2022, time.May, 15, 1, 30, 0, 0, time.UTC), // Entry.
		time.Date(2022, time.May, 10, 0, 0, 0, 0, time.UTC),  // Start.
		time.Date(2022, time.May, 20, 0, 0, 0, 0, time.UTC),  // End.
	}
	in := types.Insulin{
		Time:   times[0],
		Type:   "testType",
		Amount: 10,
	}

	replaced, err := suite.ms.WriteInsulin(ctx, &in)
	assert.NoError(suite.T(), err, "unable to write insulin to test db")
	assert.False(suite.T(), replaced, "not unique entry")

	ins, err := suite.ms.ReadInsulin(ctx, times[1], times[2])
	assert.NoError(suite.T(), err, "unable to read insulin from test db")
	assert.Len(suite.T(), ins, 1, "did not find exactly one entry")
}

func (suite *MongoTestSuite) TestReadWriteCarbsIntegration() {
	ctx := context.Background()
	times := []time.Time{
		time.Date(2022, time.May, 15, 1, 30, 0, 0, time.UTC), // Entry.
		time.Date(2022, time.May, 10, 0, 0, 0, 0, time.UTC),  // Start.
		time.Date(2022, time.May, 20, 0, 0, 0, 0, time.UTC),  // End.
	}
	carb := types.Carb{
		Time:   times[0],
		Amount: 10,
	}

	replaced, err := suite.ms.WriteCarbs(ctx, &carb)
	assert.NoError(suite.T(), err, "unable to write carbs to test db")
	assert.False(suite.T(), replaced, "not unique entry")

	carbs, err := suite.ms.ReadCarbs(ctx, times[1], times[2])
	assert.NoError(suite.T(), err, "unable to read insulin from test db")
	assert.Len(suite.T(), carbs, 1, "did not find exactly one entry")
}