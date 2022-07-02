package dexcom

import (
	"context"
	"iv2/gourgeist/defs"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
	"gopkg.in/h2non/gock.v1"
)

type DexcomTestSuite struct {
	suite.Suite
	dexcom *Client
}

func TestDexcomTestSuite(t *testing.T) {
	suite.Run(t, new(DexcomTestSuite))
}

func (suite *DexcomTestSuite) SetupSuite() {
	suite.dexcom = New("testAccount", "testPassword", zap.New(nil))
}

func (suite *DexcomTestSuite) AfterTest(_, _ string) {
	gock.Off()
}

func (suite *DexcomTestSuite) TestCreateSession() {
	gock.New(baseUrl).
		Post("/" + loginEndpoint).
		MatchType("json").
		JSON(map[string]string{
			"accountName":   "testAccount",
			"password":      "testPassword",
			"applicationId": appID,
		}).
		Reply(200).
		BodyString("test")

	client := New("testAccount", "testPassword", zap.New(nil))
	sid, err := client.CreateSession(context.Background())
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "test", sid)
}

func (suite *DexcomTestSuite) TestGetReadings() {
	expectedTrs := []*defs.TransformedReading{
		{
			Time:  time.Unix(int64(1651987807000/1000), 0),
			Mmol:  float64(219) / 18,
			Trend: "Flat",
		},
		{
			Time:  time.Unix(int64(1651988108000/1000), 0),
			Mmol:  float64(220) / 18,
			Trend: "Flat",
		},
	}

	gock.New(baseUrl).
		Post("/" + loginEndpoint).
		MatchType("json").
		JSON(map[string]string{
			"accountName":   "testAccount",
			"password":      "testPassword",
			"applicationId": appID,
		}).
		Reply(200).
		BodyString("test")

	gock.New(baseUrl).
		Get("/" + readingsEndpoint).
		MatchParams(map[string]string{
			"sessionId": "test",
			"maxCount":  "2",
		}).
		Reply(200).
		BodyString(
			`[{"WT":"Date(1651987807000)","ST":"Date(1651987807000)","DT":"Date(1651987807000-0400)","Value":219,"Trend":"Flat"},
				{"WT":"Date(1651988108000)","ST":"Date(1651988108000)","DT":"Date(1651988108000-0400)","Value":220,"Trend":"Flat"}]`,
		)

	client := New("testAccount", "testPassword", zap.New(nil))
	trs, err := client.Readings(context.Background(), 1440, 288)
	assert.NoError(suite.T(), err)
	assert.EqualValues(suite.T(), expectedTrs, trs)
}
