package discgo

import (
	"iv2/gourgeist/defs"
	"strings"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/utils/sendpart"
)

// marshalSendData transforms data of type defs.MessageData to api.SendMessageData
// which arikawa expects.
func marshalSendData(data defs.MessageData) api.SendMessageData {
	embeds := make([]discord.Embed, 0)
	for _, embed := range data.Embeds {
		fields := make([]discord.EmbedField, 0)

		for _, field := range embed.Fields {
			dField := discord.EmbedField{
				Name:   field.Name,
				Value:  field.Value,
				Inline: field.Inline,
			}
			fields = append(fields, dField)
		}

		dEmbed := discord.Embed{
			Title:       embed.Title,
			Description: embed.Description,
			Fields:      fields,
		}
		if embed.Image != nil {
			URL := embed.Image.Filename
			if !strings.Contains(embed.Image.Filename, "https://cdn.discordapp.com/attachments") {
				URL = "attachment://" + URL
			}
			dEmbed.Image = &discord.EmbedImage{URL: URL}
		}

		embeds = append(embeds, dEmbed)
	}

	files := make([]sendpart.File, 0)
	for _, file := range data.Files {
		files = append(files, sendpart.File{
			Name:   file.Name,
			Reader: file.Reader,
		})
	}

	md := api.SendMessageData{
		Content: data.Content,
		Embeds:  embeds,
		Files:   files,
	}

	if data.MentionEveryone {
		md.AllowedMentions = &api.AllowedMentions{
			Parse: []api.AllowedMentionType{api.AllowEveryoneMention},
		}
	}

	return md
}

// unmarshalMessage transforms data of type discord.Message to defs.MessageData.
func unmarshalMessage(data discord.Message) defs.MessageData {
	embeds := make([]defs.EmbedData, 0)
	for _, embed := range data.Embeds {
		fields := make([]defs.EmbedField, 0)

		for _, field := range embed.Fields {
			dField := defs.EmbedField{
				Name:   field.Name,
				Value:  field.Value,
				Inline: field.Inline,
			}
			fields = append(fields, dField)
		}

		dEmbed := defs.EmbedData{
			Title:       embed.Title,
			Description: embed.Description,
			Fields:      fields,
		}
		if embed.Image != nil {
			dEmbed.Image = &defs.ImageData{
				Filename: strings.ReplaceAll(embed.Image.URL, "attachment://", ""),
			}
		}

		embeds = append(embeds, dEmbed)
	}

	md := defs.MessageData{
		Content: data.Content,
		Embeds:  embeds,
	}

	return md
}
