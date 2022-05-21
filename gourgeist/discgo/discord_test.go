package discgo

import (
	"iv2/gourgeist/dexcom"
	"os"
	"testing"
	"time"

	"github.com/diamondburned/arikawa/api"
	"github.com/diamondburned/arikawa/discord"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
)

type DiscordTestSuite struct {
	suite.Suite
	discgo *Discord
	gid    discord.GuildID
}

func (suite *DiscordTestSuite) SetupSuite() {
	token, exist := os.LookupEnv("DISCORD_TOKEN")
	if !exist {
		panic("no token found")
	}
	discgo, err := New(token, zap.NewExample())
	if err != nil {
		panic(err)
	}
	suite.discgo = discgo
}

func (suite *DiscordTestSuite) BeforeTest(_, _ string) {
	suite.T().Log("setup test guild")
	guild, err := suite.discgo.Session.CreateGuild(api.CreateGuildData{Name: "test"})
	assert.NoError(suite.T(), err, "unable to create test guild")
	suite.gid = guild.ID

	err = suite.discgo.Setup(suite.gid.String())
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

func (suite *DiscordTestSuite) TestUpdateMainIntegration() {
	tr := dexcom.TransformedReading{
		Time:  time.Date(2022, time.May, 15, 1, 30, 0, 0, time.UTC),
		Mmol:  6.5,
		Trend: "Flat",
	}
	assert.NoError(suite.T(), suite.discgo.UpdateMain(&tr, "", nil), "unable to update main")
}

func TestDiscordTestSuiteIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	suite.Run(t, new(DiscordTestSuite))
}
