package gourgeist

import (
	"context"
	"fmt"
	"iv2/gourgeist/defs"
	dcr "iv2/gourgeist/pkg/desc"
	"iv2/gourgeist/pkg/discgo"
	"iv2/gourgeist/pkg/ghastly"
	"iv2/gourgeist/pkg/mg"
	"iv2/gourgeist/pkg/stats"
	"strconv"
	"time"

	"go.uber.org/zap"
)

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
	Descriptor    *dcr.Descriptor
	Location      *time.Location
	GlucoseConfig defs.GlucoseConfig
}

func (pu PlotUpdater) Update() error {
	end := time.Now()
	start := end.Add(defs.LookbackInterval)
	ctx := context.Background()

	glucose, err := pu.Store.ReadGlucose(ctx, start, end)
	if err != nil {
		return err
	}

	insulin, err := pu.Store.ReadInsulin(ctx, start, end)
	if err != nil {
		return err
	}

	carbs, err := pu.Store.ReadCarbs(ctx, start, end)
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

	fr, err := pu.Plotter.GenerateDailyPlot(ctx, start, end)
	if err != nil {
		pu.Logger.Debug("unable to generate daily plot", zap.Error(err))
	}

	fileReader, err := pu.Store.ReadFile(ctx, fr.GetId())
	if err != nil {
		pu.Logger.Debug("unable to read file", zap.Error(err))
	}

	if err := pu.Store.DeleteFile(ctx, fr.GetId()); err != nil {
		pu.Logger.Debug("unable to delete file", zap.Error(err))
	}

	ra := stats.TimeSpentInRange(glucose, pu.GlucoseConfig.Low, pu.GlucoseConfig.High)

	embed := defs.EmbedData{
		Title: recentGlucose.Time.In(pu.Location).Format(discgo.TimeFormat),
		Fields: []defs.EmbedField{
			{Name: "Current", Value: strconv.FormatFloat(recentGlucose.Mmol, 'f', 2, 64), Inline: true},
			{Name: "Trend", Value: recentGlucose.Trend, Inline: true},
			defs.EmptyEmbed(),
			{Name: "In Range", Value: strconv.FormatFloat(ra.InRange, 'f', 2, 64), Inline: true},
			{Name: "Above Range", Value: strconv.FormatFloat(ra.AboveRange, 'f', 2, 64), Inline: true},
			defs.EmptyEmbed(),
		},
	}

	desc, err := pu.Descriptor.New(insulin, carbs)
	if err == nil {
		embed.Description = desc
	}

	msgData := defs.MessageData{
		Embeds: []defs.EmbedData{embed},
		Files:  []defs.FileData{},
	}

	if fileReader != nil {
		pu.Logger.Debug("adding image to embed", zap.String("name", fr.GetName()))
		msgData.Embeds[0].Image = &defs.ImageData{Filename: fr.GetName()}
		msgData.Files = append(msgData.Files, defs.FileData{
			Name:   fr.GetName(),
			Reader: fileReader},
		)
	}

	return pu.Messager.NewMainMessage(msgData)
}
