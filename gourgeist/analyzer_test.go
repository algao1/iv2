package gourgeist

import (
	"context"
	"io/ioutil"
	"iv2/gourgeist/defs"
	"iv2/gourgeist/mg"
	"iv2/gourgeist/mocks"
	"testing"
	"time"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
)

const (
	testDB = "test"
)

type AnalyzerSuite struct {
	suite.Suite
	analyzer *Analyzer
	msger    *mocks.Messager
	ms       *mg.MongoStore
}

func TestAnalyzerWithTestDB(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	suite.Run(t, new(AnalyzerSuite))
}

func (suite *AnalyzerSuite) SetupSuite() {
	// TODO: This is improper testing behaviour, I'll get back to it.
	file, err := ioutil.ReadFile("../config.yaml")
	if err != nil {
		panic(err)
	}

	config := defs.Config{}
	if err = yaml.Unmarshal(file, &config); err != nil {
		panic(err)
	}

	ms, err := mg.New(context.Background(), config.Mongo, testDB, zap.NewExample())
	if err != nil {
		panic(err)
	}

	msger := &mocks.Messager{Channels: make(map[string][]discord.Message)}
	an := Analyzer{
		Messager:      msger,
		Store:         ms,
		Logger:        zap.NewExample(),
		Location:      time.Local,
		GlucoseConfig: config.Glucose,
	}
	suite.analyzer = &an
	suite.msger = msger
	suite.ms = ms
}

func (suite *AnalyzerSuite) AfterTest(_, _ string) {
	suite.T().Log("teardown test db")
	assert.NoError(
		suite.T(),
		suite.ms.Client.Database(testDB).Drop(context.Background()),
		"unable to drop test db",
	)
	suite.msger.Channels = make(map[string][]discord.Message)
}

// TODO: Maybe address code duplication here?
func (suite *AnalyzerSuite) TestGlucoseAlerts() {
	ctx := context.Background()
	_, err := suite.ms.WriteGlucose(ctx, &defs.TransformedReading{
		Time: time.Now().Add(-15 * time.Minute),
		Mmol: suite.analyzer.GlucoseConfig.Low - 1,
	})
	assert.NoError(suite.T(), err)

	assert.NoError(suite.T(), suite.analyzer.AnalyzeGlucose())
	assert.Len(suite.T(), suite.msger.Channels[alertsChannel], 1)

	alert := suite.msger.Channels[alertsChannel][0]
	label := "⚠️ " + LowGlucoseLabel
	assert.Equal(suite.T(), len(alert.Embeds), 1)
	assert.Equal(suite.T(), len(alert.Embeds[0].Fields), 1)
	assert.Equal(suite.T(), alert.Embeds[0].Fields[0].Name, label)
}

func (suite *AnalyzerSuite) TestHighGlucoseAlert() {
	ctx := context.Background()
	_, err := suite.ms.WriteGlucose(ctx, &defs.TransformedReading{
		Time: time.Now().Add(-15 * time.Minute),
		Mmol: suite.analyzer.GlucoseConfig.High + 1,
	})
	assert.NoError(suite.T(), err)

	assert.NoError(suite.T(), suite.analyzer.AnalyzeGlucose())
	assert.Len(suite.T(), suite.msger.Channels[alertsChannel], 1)

	alert := suite.msger.Channels[alertsChannel][0]
	label := "⚠️ " + HighGlucoseLabel
	assert.Equal(suite.T(), len(alert.Embeds), 1)
	assert.Equal(suite.T(), len(alert.Embeds[0].Fields), 1)
	assert.Equal(suite.T(), alert.Embeds[0].Fields[0].Name, label)
}
