package commander

import (
	"context"
	"fmt"
	"iv2/gourgeist/defs"
	"iv2/gourgeist/pkg/ghastly"
	"iv2/gourgeist/pkg/ghastly/proto"
	"iv2/gourgeist/pkg/stats"
	"strconv"
	"time"

	"go.uber.org/zap"
)

func handleGenReport(cs CommanderStore, cd CommanderDisplay, p ghastly.Plotter,
	gcfg defs.GlucoseConfig, logger *zap.Logger, loc *time.Location, data defs.CommandInteraction) error {
	timeframe := data.Options[0].Value
	offset, _ := strconv.Atoi(data.Options[1].Value)

	start := time.Now().In(loc)
	var end time.Time

	var fr *proto.FileResponse
	var err error

	switch timeframe {
	case "w":
		start = startOfWeek(start)
		start = start.AddDate(0, 0, -int(offset)*7)
		end = start.AddDate(0, 0, 7)

		fr, err = p.GenerateWeeklyPlot(context.Background(), start, end)
		if err != nil {
			logger.Debug("unable to generate weekly plot", zap.Error(err))
		}
	case "m":
		start = startOfMonth(start.AddDate(0, -int(offset), 0))
		start = start.AddDate(0, 0, 1)
		end = start.AddDate(0, 1, 0)
	}

	fileReader, err := cs.ReadFile(context.Background(), fr.GetId())
	if err != nil {
		logger.Debug("unable to read file", zap.Error(err))
	}

	if err := cs.DeleteFile(context.Background(), fr.GetId()); err != nil {
		logger.Debug("unable to delete file", zap.Error(err))
	}

	glucose, err := cs.ReadGlucose(context.Background(), start, end)
	if err != nil {
		return err
	}

	insulin, err := cs.ReadInsulin(context.Background(), start, end)
	if err != nil {
		return err
	}

	carbs, err := cs.ReadCarbs(context.Background(), start, end)
	if err != nil {
		return err
	}

	ra := stats.TimeSpentInRange(glucose, gcfg.Low, gcfg.High)
	ss := stats.GlucoseSummary(glucose)
	dd := stats.DailyAggregate(stats.IntakeData{Ins: insulin, Carbs: carbs}, loc)

	var desc string
	desc += fmt.Sprintf(
		"%5s %6s %6s %6s\n",
		"",
		defs.RapidActing.String(),
		defs.SlowActing.String(),
		"carbs",
	)
	for _, day := range dd.Days {
		var rSum, sSum float64
		for _, v := range dd.InsMap[day] {
			switch v.Type {
			case defs.RapidActing.String():
				rSum += v.Amount
			case defs.SlowActing.String():
				sSum += v.Amount
			}
		}

		var cSum float64
		for _, v := range dd.CarbsMap[day] {
			cSum += v.Amount
		}

		desc += fmt.Sprintf("%s %6.f %6.f %6.f\n", day.Format(monthDayFormat), rSum, sSum, cSum)
	}

	msgData := defs.MessageData{
		Embeds: []defs.EmbedData{
			{
				Title:       fmt.Sprintf("%s to %s", start.Format(monthDayFormat), end.Format(monthDayFormat)),
				Description: "```" + desc + "```",
				Fields: []defs.EmbedField{
					{Name: "Average", Value: strconv.FormatFloat(ss.Average, 'f', 2, 64), Inline: true},
					{Name: "Deviation", Value: strconv.FormatFloat(ss.Deviation, 'f', 2, 64), Inline: true},
					defs.EmptyEmbed(),
					{Name: "In Range", Value: strconv.FormatFloat(ra.InRange, 'f', 2, 64), Inline: true},
					{Name: "Above Range", Value: strconv.FormatFloat(ra.AboveRange, 'f', 2, 64), Inline: true},
					defs.EmptyEmbed(),
				},
			},
		},
	}

	if fileReader != nil {
		logger.Debug("adding image to embed", zap.String("name", fr.GetName()))
		msgData.Embeds[0].Image = &defs.ImageData{Filename: fr.GetName()}
		msgData.Files = append(msgData.Files, defs.FileData{Name: fr.GetName(), Reader: fileReader})
	}

	_, err = cd.SendMessage(msgData, defs.ReportsChannel)
	return err
}

func startOfWeek(t time.Time) time.Time {
	if wd := t.Weekday(); wd == time.Sunday {
		t = t.AddDate(0, 0, -6)
	} else {
		t = t.AddDate(0, 0, -int(wd)+1)
	}
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

func startOfMonth(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), 0, 0, 0, 0, 0, t.Location())
}
