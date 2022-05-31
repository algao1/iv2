package discgo

import (
	"iv2/gourgeist/dexcom"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/utils/json/option"
	"github.com/diamondburned/arikawa/v3/utils/sendpart"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
)

type DiscordTestSuite struct {
	suite.Suite
	discgo *Discord
	gid    discord.GuildID
}

func TestDiscordTestSuiteIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	suite.Run(t, new(DiscordTestSuite))
}

func (suite *DiscordTestSuite) SetupSuite() {
	token, exist := os.LookupEnv("DISCORD_TOKEN")
	if !exist {
		panic("no token found")
	}
	discgo, err := New(token, InteractionCreateHandler, zap.NewExample(), time.Local)
	if err != nil {
		panic(err)
	}
	suite.discgo = discgo

	var wg sync.WaitGroup

	guilds, err := suite.discgo.Session.Guilds(10)
	if err != nil {
		panic(err)
	}

	// Delete uncleared test guilds.
	for _, guild := range guilds {
		if guild.Name == "test" {
			wg.Add(1)
			go suite.discgo.Session.DeleteGuild(guild.ID)
		}
	}

	wg.Wait()
}

func (suite *DiscordTestSuite) BeforeTest(_, _ string) {
	suite.T().Log("setup test guild")
	guild, err := suite.discgo.Session.CreateGuild(api.CreateGuildData{Name: "test"})
	assert.NoError(suite.T(), err, "unable to create test guild")
	suite.gid = guild.ID

	err = suite.discgo.Setup(suite.gid.String(), false)
	assert.NoError(suite.T(), err, "unable to complete setup")
}

func (suite *DiscordTestSuite) AfterTest(_, _ string) {
	suite.T().Log("teardown test guild")
	err := suite.discgo.Session.DeleteGuild(suite.gid)
	assert.NoError(suite.T(), err, "unable to delete test guild")
}

func (suite *DiscordTestSuite) TestSetupIntegration() {
	channels, err := suite.discgo.Session.Channels(suite.gid)
	assert.NoError(suite.T(), err, "unable to get channels")

	var chFound bool
	for _, ch := range channels {
		if ch.Name == broadcastChannelName {
			chFound = true
		}
	}
	assert.True(suite.T(), chFound, "broadcast channel not found")
}

func getSimpleMessageData() api.SendMessageData {
	tr := dexcom.TransformedReading{
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
