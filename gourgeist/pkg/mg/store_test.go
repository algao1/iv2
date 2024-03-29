package mg

import (
	"context"
	"io/ioutil"
	"iv2/gourgeist/defs"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
)

const testDB = "test"

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
	file, err := ioutil.ReadFile("../../../config.yaml")
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
	col := "test"
	doc := defs.Insulin{ID: defs.MyObjectID(id.Hex())}

	var fetchedDoc defs.Insulin
	_, err := suite.ms.InsertNew(ctx, col, &doc)
	assert.NoError(suite.T(), err)
	assert.NoError(suite.T(),
		suite.ms.DocByID(ctx, col, id.Hex(), &fetchedDoc),
		"unable to fetch document by id",
	)
	assert.EqualValues(suite.T(), doc, fetchedDoc, "not same document")
}

func (suite *MongoTestSuite) TestDeleteByIDIntegration() {
	ctx := context.Background()
	id := primitive.NewObjectID()
	col := "test"
	doc := defs.Insulin{ID: defs.MyObjectID(id.Hex())}

	var fetchedDoc defs.Insulin
	_, err := suite.ms.InsertNew(ctx, col, &doc)
	assert.NoError(suite.T(), err)
	assert.NoError(suite.T(), suite.ms.DeleteByID(ctx, col, id.Hex()))
	assert.Error(suite.T(),
		suite.ms.DocByID(ctx, col, id.Hex(), &fetchedDoc),
		"found document by id, delete not successful",
	)
}

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
		assert.Equal(suite.T(), int64(0), res.MatchedCount, "not unique entry")
	}

	trs, err := suite.ms.ReadGlucose(ctx, times[2], times[3])
	assert.NoError(suite.T(), err, "unable to read glucose from test db")
	assert.Len(suite.T(), trs, len(trsInsert), "did not find exactly one entry")
	for i := range trs {
		assert.NotEmpty(suite.T(), trs[i].ID)
		assert.EqualValues(suite.T(), trsInsert[i].Mmol, trs[i].Mmol)
		assert.EqualValues(suite.T(), trsInsert[i].Time, trs[i].Time)
		assert.EqualValues(suite.T(), trsInsert[i].Trend, trs[i].Trend)
	}
}

func (suite *MongoTestSuite) TestIgnoreDupeInsertIntegration() {
	ctx := context.Background()
	tr := defs.TransformedReading{
		Time:  time.Date(2022, time.May, 12, 1, 30, 0, 0, time.UTC),
		Mmol:  6.5,
		Trend: "Flat",
	}

	res, err := suite.ms.WriteGlucose(ctx, &tr)
	assert.NoError(suite.T(), err, "unable to write glucose to test db")
	assert.Equal(suite.T(), int64(0), res.MatchedCount, "not unique entry")

	res, err = suite.ms.WriteGlucose(ctx, &tr)
	assert.NoError(suite.T(), err, "unable to write glucose to test db")
	assert.Equal(suite.T(), int64(1), res.MatchedCount)
	assert.Equal(suite.T(), int64(0), res.UpsertedCount)
	assert.Equal(suite.T(), int64(0), res.ModifiedCount)
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
		assert.Equal(suite.T(), int64(0), res.MatchedCount, "not unique entry")
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

func (suite *MongoTestSuite) TestUpdateInsulinIntegration() {
	ctx := context.Background()
	in := defs.Insulin{
		Time:   time.Date(2022, time.May, 12, 1, 30, 0, 0, time.UTC),
		Type:   "testType",
		Amount: 10,
	}

	res, err := suite.ms.WriteInsulin(ctx, &in)
	assert.NoError(suite.T(), err, "unable to write insulin to test db")
	assert.Equal(suite.T(), int64(0), res.MatchedCount, "not unique entry")

	in.ID, in.Amount = res.UpsertedID, 42
	ures, err := suite.ms.UpdateInsulin(ctx, &in)
	assert.NoError(suite.T(), err, "unable to update insulin")
	assert.Equal(suite.T(), int64(1), ures.ModifiedCount)

	var updatedIn defs.Insulin
	assert.NoError(suite.T(), suite.ms.DocByID(ctx, InsulinCollection, string(res.UpsertedID), &updatedIn))
	assert.EqualValues(suite.T(), in, updatedIn)
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
		assert.Equal(suite.T(), int64(0), res.MatchedCount, "not unique entry")
	}

	carbs, err := suite.ms.ReadCarbs(ctx, times[2], times[3])
	assert.NoError(suite.T(), err, "unable to read carbs from test db")
	assert.Len(suite.T(), carbs, len(carbsInsert), "did not find exactly one entry")
	for i := range carbs {
		assert.EqualValues(suite.T(), carbsInsert[i].Amount, carbs[i].Amount)
		assert.EqualValues(suite.T(), carbsInsert[i].Time, carbs[i].Time)
	}
}

func (suite *MongoTestSuite) TestUpdateCarbsIntegration() {
	ctx := context.Background()
	carbs := defs.Carb{
		Time:   time.Date(2022, time.May, 12, 1, 30, 0, 0, time.UTC),
		Amount: 10,
	}

	res, err := suite.ms.WriteCarbs(ctx, &carbs)
	assert.NoError(suite.T(), err, "unable to write carbs to test db")
	assert.Equal(suite.T(), int64(0), res.MatchedCount, "not unique entry")

	carbs.ID, carbs.Amount = res.UpsertedID, 42
	ures, err := suite.ms.UpdateCarbs(ctx, &carbs)
	assert.NoError(suite.T(), err, "unable to update carbs")
	assert.Equal(suite.T(), int64(1), ures.ModifiedCount)

	var updatedCarbs defs.Carb
	assert.NoError(suite.T(), suite.ms.DocByID(ctx, CarbsCollection, string(res.UpsertedID), &updatedCarbs))
	assert.EqualValues(suite.T(), carbs, updatedCarbs)
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
		assert.Equal(suite.T(), int64(0), res.MatchedCount, "not unique entry")
	}

	alerts, err := suite.ms.ReadAlerts(ctx, times[2], times[3])
	assert.NoError(suite.T(), err, "unable to read alerts from test db")
	assert.Len(suite.T(), alerts, len(alertsInsert), "did not find exactly one entry")
	for i := range alerts {
		assert.EqualValues(suite.T(), alertsInsert[i].Label, alerts[i].Label)
		assert.EqualValues(suite.T(), alertsInsert[i].Reason, alerts[i].Reason)
	}
}
