package mg

import (
	"context"
	"io/ioutil"
	"iv2/gourgeist/defs"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
)

const (
	mongoURI = "mongodb://localhost:27017"
	testDB   = "test"
)

type MongoTestSuite struct {
	suite.Suite
	ms *MongoStore
}

func TestMongoIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	suite.Run(t, new(MongoTestSuite))
}

func (suite *MongoTestSuite) SetupSuite() {
	// TODO: This is improper testing behaviour, I'll get back to it.
	file, err := ioutil.ReadFile("../../config.yaml")
	if err != nil {
		panic(err)
	}

	config := defs.Config{}
	if err = yaml.Unmarshal(file, &config); err != nil {
		panic(err)
	}

	ms, err := New(context.Background(), config.Mongo, testDB, zap.NewExample())
	if err != nil {
		panic(err)
	}
	suite.ms = ms

	assert.NoError(
		suite.T(),
		suite.ms.Client.Database(testDB).Drop(context.Background()),
		"unable to drop test db",
	)
}

func (suite *MongoTestSuite) AfterTest(_, _ string) {
	suite.T().Log("teardown test db")
	assert.NoError(
		suite.T(),
		suite.ms.Client.Database(testDB).Drop(context.Background()),
		"unable to drop test db",
	)
}

func (suite *MongoTestSuite) TestDocByIDIntegration() {
	ctx := context.Background()
	id := primitive.NewObjectID()
	doc := defs.Insulin{ID: &id}

	var fetchedDoc defs.Insulin
	_, err := suite.ms.Upsert(ctx, "test", bson.M{}, &doc)
	assert.NoError(suite.T(), err)
	assert.NoError(suite.T(), suite.ms.DocByID(ctx, "test", &id, &fetchedDoc), "unable to fetch document by id")
	assert.EqualValues(suite.T(), doc, fetchedDoc, "not same document")
}

func (suite *MongoTestSuite) TestDeleteByIDIntegration() {
	ctx := context.Background()
	id := primitive.NewObjectID()
	doc := defs.Insulin{ID: &id}

	var fetchedDoc defs.Insulin
	_, err := suite.ms.Upsert(ctx, "test", bson.M{}, &doc)
	assert.NoError(suite.T(), err)
	assert.NoError(suite.T(), suite.ms.DeleteByID(ctx, "test", &id))
	assert.Error(suite.T(),
		suite.ms.DocByID(ctx, "test", &id, &fetchedDoc),
		"found document by id, delete not successful",
	)
}

// TODO: Clean up these unit tests to reduce duplicate code.

func (suite *MongoTestSuite) TestRWGlucoseIntegration() {
	ctx := context.Background()
	times := []time.Time{
		time.Date(2022, time.May, 12, 1, 30, 0, 0, time.UTC),
		time.Date(2022, time.May, 15, 1, 30, 0, 0, time.UTC),
		time.Date(2022, time.May, 10, 0, 0, 0, 0, time.UTC), // Start.
		time.Date(2022, time.May, 20, 0, 0, 0, 0, time.UTC), // End.
	}
	trsInsert := []defs.TransformedReading{
		{
			Time:  times[0],
			Mmol:  6.5,
			Trend: "Flat",
		},
		{
			Time:  times[1],
			Mmol:  7.2,
			Trend: "Flat",
		},
	}

	for _, tr := range trsInsert {
		res, err := suite.ms.WriteGlucose(ctx, &tr)
		assert.NoError(suite.T(), err, "unable to write glucose to test db")
		assert.True(suite.T(), res.MatchedCount == 0, "not unique entry")
	}

	trs, err := suite.ms.ReadGlucose(ctx, times[2], times[3])
	assert.NoError(suite.T(), err, "unable to read glucose from test db")
	assert.Len(suite.T(), trs, len(trsInsert), "did not find exactly one entry")
	for i := range trs {
		assert.EqualValues(suite.T(), trsInsert[i].Mmol, trs[i].Mmol)
		assert.EqualValues(suite.T(), trsInsert[i].Time, trs[i].Time)
		assert.EqualValues(suite.T(), trsInsert[i].Trend, trs[i].Trend)
	}
}

func (suite *MongoTestSuite) TestRWInsulinIntegration() {
	ctx := context.Background()
	times := []time.Time{
		time.Date(2022, time.May, 12, 1, 30, 0, 0, time.UTC),
		time.Date(2022, time.May, 15, 1, 30, 0, 0, time.UTC),
		time.Date(2022, time.May, 10, 0, 0, 0, 0, time.UTC), // Start.
		time.Date(2022, time.May, 20, 0, 0, 0, 0, time.UTC), // End.
	}
	insInsert := []defs.Insulin{
		{
			Time:   times[0],
			Type:   "testType",
			Amount: 10,
		},
		{
			Time:   times[1],
			Type:   "testType",
			Amount: 10,
		},
	}

	for _, in := range insInsert {
		res, err := suite.ms.WriteInsulin(ctx, &in)
		assert.NoError(suite.T(), err, "unable to write insulin to test db")
		assert.True(suite.T(), res.MatchedCount == 0, "not unique entry")
	}

	ins, err := suite.ms.ReadInsulin(ctx, times[2], times[3])
	assert.NoError(suite.T(), err, "unable to read insulin from test db")
	assert.Len(suite.T(), ins, len(insInsert), "did not find all entries")
	for i := range ins {
		assert.EqualValues(suite.T(), insInsert[i].Amount, ins[i].Amount)
		assert.EqualValues(suite.T(), insInsert[i].Time, ins[i].Time)
		assert.EqualValues(suite.T(), insInsert[i].Type, ins[i].Type)
	}
}

func (suite *MongoTestSuite) TestRWCarbsIntegration() {
	ctx := context.Background()
	times := []time.Time{
		time.Date(2022, time.May, 12, 1, 30, 0, 0, time.UTC),
		time.Date(2022, time.May, 15, 1, 30, 0, 0, time.UTC),
		time.Date(2022, time.May, 10, 0, 0, 0, 0, time.UTC), // Start.
		time.Date(2022, time.May, 20, 0, 0, 0, 0, time.UTC), // End.
	}
	carbsInsert := []defs.Carb{
		{
			Time:   times[0],
			Amount: 10,
		},
		{
			Time:   times[1],
			Amount: 10,
		},
	}

	for _, carb := range carbsInsert {
		res, err := suite.ms.WriteCarbs(ctx, &carb)
		assert.NoError(suite.T(), err, "unable to write carbs to test db")
		assert.True(suite.T(), res.MatchedCount == 0, "not unique entry")
	}

	carbs, err := suite.ms.ReadCarbs(ctx, times[2], times[3])
	assert.NoError(suite.T(), err, "unable to read carbs from test db")
	assert.Len(suite.T(), carbs, len(carbsInsert), "did not find exactly one entry")
	for i := range carbs {
		assert.EqualValues(suite.T(), carbsInsert[i].Amount, carbs[i].Amount)
		assert.EqualValues(suite.T(), carbsInsert[i].Time, carbs[i].Time)
	}
}

func (suite *MongoTestSuite) TestRWAlertsIntegration() {
	ctx := context.Background()
	times := []time.Time{
		time.Date(2022, time.May, 12, 1, 30, 0, 0, time.UTC),
		time.Date(2022, time.May, 15, 1, 30, 0, 0, time.UTC),
		time.Date(2022, time.May, 10, 0, 0, 0, 0, time.UTC), // Start.
		time.Date(2022, time.May, 20, 0, 0, 0, 0, time.UTC), // End.
	}
	alertsInsert := []defs.Alert{
		{
			Time:   times[0],
			Label:  "testlabel",
			Reason: "testreason",
		},
		{
			Time:   times[1],
			Label:  "testlabel2",
			Reason: "testreason2",
		},
	}

	for _, alert := range alertsInsert {
		res, err := suite.ms.WriteAlert(ctx, &alert)
		assert.NoError(suite.T(), err, "unable to write alerts to test db")
		assert.True(suite.T(), res.MatchedCount == 0, "not unique entry")
	}

	alerts, err := suite.ms.ReadAlerts(ctx, times[2], times[3])
	assert.NoError(suite.T(), err, "unable to read alerts from test db")
	assert.Len(suite.T(), alerts, len(alertsInsert), "did not find exactly one entry")
	for i := range alerts {
		assert.EqualValues(suite.T(), alertsInsert[i].Label, alerts[i].Label)
		assert.EqualValues(suite.T(), alertsInsert[i].Reason, alerts[i].Reason)
	}
}
