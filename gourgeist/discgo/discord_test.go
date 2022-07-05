package discgo

import (
	"io/ioutil"
	"iv2/gourgeist/defs"
	"strconv"
	"testing"
	"time"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/utils/json/option"
	"github.com/diamondburned/arikawa/v3/utils/sendpart"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
)

const (
	testChannel = "test"
)

type DiscordTestSuite struct {
	suite.Suite
	discgo *Discord
}

func TestDiscordTestSuiteIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	suite.Run(t, new(DiscordTestSuite))
}

func (suite *DiscordTestSuite) SetupSuite() {
	// TODO: This is improper testing behaviour, I'll get back to it.
	file, err := ioutil.ReadFile("../../config.yaml")
	if err != nil {
		panic(err)
	}

	config := defs.Config{}
	if err = yaml.Unmarshal(file, &config); err != nil {
		panic(err)
	}

	discgo, err := New(
		config.Discord.Token,
		strconv.Itoa(config.Discord.Guild),
		zap.NewExample(),
		time.Local,
	)
	if err != nil {
		panic(err)
	}

	// TODO: Need to redo this eventually, not a good system.
	discgo.mainCh = testChannel
	suite.discgo = discgo
}

func (suite *DiscordTestSuite) BeforeTest(_, _ string) {
	err := suite.discgo.Setup([]api.CreateCommandData{}, []string{})
	assert.NoError(suite.T(), err, "unable to complete setup")
}

func (suite *DiscordTestSuite) AfterTest(_, _ string) {
	for name, id := range suite.discgo.channels {
		if name == testChannel {
			err := suite.discgo.Session.DeleteChannel(id, api.AuditLogReason("delete test channel"))
			assert.NoError(suite.T(), err, "unable to delete test channel")
		}
	}
}

func (suite *DiscordTestSuite) TestSetupIntegration() {
	channels, err := suite.discgo.Session.Channels(suite.discgo.gid)
	assert.NoError(suite.T(), err, "unable to get channels")

	var chFound bool
	for _, ch := range channels {
		if ch.Name == testChannel {
			chFound = true
		}
	}
	assert.True(suite.T(), chFound, "broadcast channel not found")
}

func getSimpleMessageData() api.SendMessageData {
	tr := defs.TransformedReading{
		Time:  time.Date(2022, time.May, 15, 1, 30, 0, 0, time.UTC),
		Mmol:  6.5,
		Trend: "Flat",
	}

	embed := discord.Embed{
		Title: tr.Time.In(time.UTC).Format(TimeFormat),
		Fields: []discord.EmbedField{
			{Name: "Current", Value: strconv.FormatFloat(tr.Mmol, 'f', 2, 64)},
		},
	}

	return api.SendMessageData{
		Embeds: []discord.Embed{embed},
		Files:  []sendpart.File{},
	}
}

func (suite *DiscordTestSuite) TestNewMainIntegration() {
	msgData := getSimpleMessageData()
	assert.NoError(suite.T(), suite.discgo.NewMainMessage(msgData), "unable to send main")
}

func (suite *DiscordTestSuite) TestOverwriteMainIntegration() {
	msgData := getSimpleMessageData()
	assert.NoError(suite.T(), suite.discgo.NewMainMessage(msgData), "unable to send main")
	assert.NoError(suite.T(), suite.discgo.NewMainMessage(msgData), "unable to send main")
}

func (suite *DiscordTestSuite) TestGetMainIntegration() {
	msgData := getSimpleMessageData()
	assert.NoError(suite.T(), suite.discgo.NewMainMessage(msgData), "unable to send main")

	msg, err := suite.discgo.GetMainMessage()
	assert.NoError(suite.T(), err, "unable to get main")
	assert.Len(suite.T(), msg.Embeds, 1, "did not find exactly one embed")
	assert.EqualValues(suite.T(), msgData.Embeds[0], msg.Embeds[0], "got different embeds")
}

func (suite *DiscordTestSuite) TestUpdateMainIntegration() {
	msgData := getSimpleMessageData()
	assert.NoError(suite.T(), suite.discgo.NewMainMessage(msgData), "unable to send main")

	editData := api.EditMessageData{
		Content: option.NewNullableString("test"),
	}
	assert.NoError(suite.T(), suite.discgo.UpdateMainMessage(editData), "unable to edit main")

	msg, err := suite.discgo.GetMainMessage()
	assert.NoError(suite.T(), err, "unable to get main")
	assert.Len(suite.T(), msg.Embeds, 1, "did not find exactly one embed")
	assert.EqualValues(suite.T(), msgData.Embeds[0], msg.Embeds[0], "got different embeds")
	assert.EqualValues(suite.T(), editData.Content.Val, msg.Content)
}
