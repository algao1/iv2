package dexcom

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"gopkg.in/h2non/gock.v1"
)

func TestCreateSession(t *testing.T) {
	defer gock.Off()

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
	assert.NoError(t, err)
	assert.Equal(t, "test", sid)
}

func TestGetReadings(t *testing.T) {
	defer gock.Off()

	expectedTrs := []*TransformedReading{
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
	assert.NoError(t, err)
	assert.EqualValues(t, expectedTrs, trs)
}
