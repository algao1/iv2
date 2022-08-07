package discgo

import (
	"iv2/gourgeist/defs"
	"testing"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type DiscordTypeTestSuite struct {
	suite.Suite
}

func TestDiscordTypeSuite(t *testing.T) {
	suite.Run(t, new(DiscordTypeTestSuite))
}

func newMessageData() defs.MessageData {
	return defs.MessageData{
		Content: "test content",
		Embeds: []defs.EmbedData{
			{
				Title:       "title1",
				Description: "description1",
				Fields: []defs.EmbedField{
					{
						Name:   "field1",
						Value:  "value1",
						Inline: false,
					},
					defs.EmptyEmbed(),
				},
				Image: &defs.ImageData{
					Filename: "testFile",
				},
			},
		},
		Files: []defs.FileData{
			{
				Name:   "testFile",
				Reader: nil,
			},
		},
		MentionEveryone: true,
	}
}

func (suite *DiscordTypeTestSuite) TestMarshalData() {
	input := newMessageData()
	output := marshalSendData(input)

	assert.Equal(suite.T(), input.Content, output.Content)
	assert.Equal(suite.T(), len(input.Embeds), len(output.Embeds))
	assert.Equal(suite.T(), len(input.Files), len(output.Files))

	// Assert embeds.
	assert.Equal(suite.T(),
		input.Embeds[0].Title,
		output.Embeds[0].Title,
	)
	assert.Equal(suite.T(),
		input.Embeds[0].Description,
		output.Embeds[0].Description,
	)
	assert.EqualValues(suite.T(), discord.EmbedField{
		Name:   "field1",
		Value:  "value1",
		Inline: false,
	}, output.Embeds[0].Fields[0])
	assert.EqualValues(suite.T(), discord.EmbedField{
		Name:   "\u200b",
		Value:  "\u200b",
		Inline: true,
	}, output.Embeds[0].Fields[1])
	assert.Equal(suite.T(),
		"attachment://"+input.Embeds[0].Image.Filename,
		output.Embeds[0].Image.URL,
	)

	// Assert mention.
	assert.Equal(suite.T(),
		api.AllowEveryoneMention,
		output.AllowedMentions.Parse[0],
	)
}

func (suite *DiscordTypeTestSuite) TestUnmarshalData() {
	input := newMessageData()
	output := marshalSendData(input)

	dmsg := discord.Message{
		Content: output.Content,
		Embeds:  output.Embeds,
	}
	unmarshalled := unmarshalMessage(dmsg)

	assert.EqualValues(suite.T(), input.Content, unmarshalled.Content)
	assert.EqualValues(suite.T(), input.Embeds, unmarshalled.Embeds)
}
