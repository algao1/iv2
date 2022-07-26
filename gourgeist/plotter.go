package gourgeist

import (
	"context"
	"fmt"
	"iv2/gourgeist/defs"
	"iv2/gourgeist/pkg/discgo"
	"iv2/gourgeist/pkg/ghastly"
	"iv2/gourgeist/pkg/mg"
	"iv2/gourgeist/pkg/stats"
	"strconv"
	"time"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/utils/sendpart"
	"go.uber.org/zap"
)

const lookbackInterval = -12 * time.Hour

var inlineBlankField = discord.EmbedField{
	Name:   "\u200b",
	Value:  "\u200b",
	Inline: true,
}

type PlotterStore interface {
	mg.GlucoseStore
	mg.InsulinStore
	mg.CarbStore
	mg.FileStore
}

// TODO: Need to rename, not only updates plots, but is responsible
// for also updating the 'main' display.
type PlotUpdater struct {
	Messager discgo.Messager
	Plotter  ghastly.Plotter
	Store    PlotterStore

	Logger        *zap.Logger
	Location      *time.Location
	GlucoseConfig defs.GlucoseConfig
}

func (pu PlotUpdater) Update() error {
	end := time.Now()
	start := end.Add(lookbackInterval)
	ctx := context.Background()

	glucose, err := pu.Store.ReadGlucose(ctx, start, end)
	if err != nil {
		return err
	}

	if len(glucose) == 0 {
		return fmt.Errorf("no glucose readings found")
	}

	prevMsg, err := pu.Messager.GetMainMessage()
	if err != nil {
		pu.Logger.Debug("unable to get main message", zap.Error(err))
	}

	recentGlucose := glucose[len(glucose)-1]
	if prevMsg != nil && len(prevMsg.Embeds) > 0 &&
		prevMsg.Embeds[0].Title == recentGlucose.Time.In(pu.Location).Format(discgo.TimeFormat) {
		pu.Logger.Debug(
			"skipping display update, up to date",
			zap.String("date", prevMsg.Embeds[0].Title),
		)
		return nil
	}

	fr, err := pu.Plotter.GenerateDailyPlot(context.Background(), start, end)
	if err != nil {
		pu.Logger.Debug("unable to generate daily plot", zap.Error(err))
	}

	fileReader, err := pu.Store.ReadFile(context.Background(), fr.GetId())
	if err != nil {
		pu.Logger.Debug("unable to read file", zap.Error(err))
	}

	if err := pu.Store.DeleteFile(context.Background(), fr.GetId()); err != nil {
		pu.Logger.Debug("unable to delete file", zap.Error(err))
	}

	ra := stats.TimeSpentInRange(glucose, pu.GlucoseConfig.Low, pu.GlucoseConfig.High)

	embed := discord.Embed{
		Title: recentGlucose.Time.In(pu.Location).Format(discgo.TimeFormat),
		Fields: []discord.EmbedField{
			{Name: "Current", Value: strconv.FormatFloat(recentGlucose.Mmol, 'f', 2, 64), Inline: true},
			{Name: "Trend", Value: recentGlucose.Trend, Inline: true},
			inlineBlankField,
			{Name: "In Range", Value: strconv.FormatFloat(ra.InRange, 'f', 2, 64), Inline: true},
			{Name: "Above Range", Value: strconv.FormatFloat(ra.AboveRange, 'f', 2, 64), Inline: true},
			inlineBlankField,
		},
	}

	desc, err := newDescription(pu.Store, pu.Location)
	if err == nil {
		embed.Description = desc
	}

	msgData := api.SendMessageData{
		Embeds: []discord.Embed{embed},
		Files:  []sendpart.File{},
	}

	if fileReader != nil {
		pu.Logger.Debug("adding image to embed", zap.String("name", fr.GetName()))
		msgData.Embeds[0].Image = &discord.EmbedImage{URL: "attachment://" + fr.GetName()}
		msgData.Files = append(msgData.Files, sendpart.File{Name: fr.GetName(), Reader: fileReader})
	}

	return pu.Messager.NewMainMessage(msgData)
}
