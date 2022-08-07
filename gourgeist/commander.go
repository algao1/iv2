package gourgeist

import (
	"context"
	"fmt"
	"iv2/gourgeist/defs"
	"iv2/gourgeist/pkg/discgo"
	"iv2/gourgeist/pkg/ghastly"
	"iv2/gourgeist/pkg/ghastly/proto"
	"iv2/gourgeist/pkg/mg"
	"iv2/gourgeist/pkg/stats"
	"reflect"
	"strconv"
	"time"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/utils/json/option"
	"go.uber.org/zap"
)

const (
	CmdTimeFormat  = "03:04 PM"
	MonthDayFormat = "01/02"
	logLimit       = 7
)

type CommanderStore interface {
	mg.DocumentStore
	mg.GlucoseStore
	mg.InsulinStore
	mg.CarbStore
	mg.FileStore
}

type CommanderDisplay interface {
	discgo.Messager
	discgo.Interactioner
}

type CommandHandler struct {
	Display CommanderDisplay
	Plotter ghastly.Plotter
	Store   CommanderStore

	Logger        *zap.Logger
	Location      *time.Location
	GlucoseConfig defs.GlucoseConfig
}

func (ch *CommandHandler) InteractionCreateHandler() func(*gateway.InteractionCreateEvent) {
	return func(e *gateway.InteractionCreateEvent) {
		switch data := e.Data.(type) {
		case *discord.CommandInteraction:
			if err := ch.handleCommand(data); err != nil {
				ch.Logger.Debug("unable to handle command",
					zap.String("command", data.Name),
					zap.Error(err),
				)
			}
		}

		resp := api.InteractionResponse{
			Type: api.MessageInteractionWithSource,
			Data: &api.InteractionResponseData{Content: option.NewNullableString("received")},
		}

		if err := ch.Display.RespondInteraction(e.ID, e.Token, resp); err != nil {
			ch.Logger.Debug("unable to send interaction callback", zap.Error(err))
			return
		}
		if err := ch.Display.DeleteInteractionResponse(e.AppID, e.Token); err != nil {
			ch.Logger.Debug("unable to delete interaction response", zap.String("token", e.Token), zap.Error(err))
		}
	}
}

func (ch *CommandHandler) handleCommand(data *discord.CommandInteraction) error {
	ch.Logger.Debug("received command",
		zap.String("cmd", data.Name),
		zap.Any("options", data.Options),
	)

	switch data.Name {
	case AddCarbsCmd:
		return ch.handleCarbs(data)
	case EditCarbsCmd:
		return ch.handleEditCarbs(data)
	case AddInsulinCmd:
		return ch.handleInsulin(data)
	case EditInsulinCmd:
		return ch.handleEditInsulin(data)
	case GenReportCmd:
		return ch.handleGenReport(data)
	default:
		return fmt.Errorf("unknown command: %s", data.Name)
	}
}

func (ch *CommandHandler) handleCarbs(data *discord.CommandInteraction) error {
	amount, _ := data.Options[0].IntValue()

	_, err := ch.Store.WriteCarbs(context.Background(), &defs.Carb{
		Time:   time.Now().In(ch.Location),
		Amount: float64(amount),
	})
	if err != nil {
		return fmt.Errorf("unable to save carbs: %w", err)
	}

	return ch.updateWithEvent()
}

func (ch *CommandHandler) handleEditCarbs(data *discord.CommandInteraction) error {
	ctx := context.Background()
	id := data.Options[0].String()

	var carb defs.Carb
	var err error
	if len(id) == 6 {
		carbs, err := ch.Store.ReadCarbs(
			ctx,
			time.Now().Add(lookbackInterval),
			time.Now(),
		)
		if err != nil {
			return err
		}

		for _, c := range carbs {
			if hashDigest(string(c.ID)) == id {
				carb = c
				break
			}
		}
		if reflect.ValueOf(carb).IsZero() {
			return fmt.Errorf("no entry found with digest %s", id)
		}
	} else {
		err := ch.Store.DocByID(ctx, mg.CarbsCollection, id, &carb)
		if err != nil {
			return err
		}
	}

	var amount = carb.Amount
	var minuteOffset int64

	for _, opt := range data.Options[1:] {
		switch opt.Name {
		case "amount":
			amount, err = opt.FloatValue()
		case "offset":
			minuteOffset, err = opt.IntValue()
		}
		if err != nil {
			return err
		}
	}

	if amount < 0 {
		if err := ch.Store.DeleteByID(ctx, mg.CarbsCollection, string(carb.ID)); err != nil {
			return err
		}
	} else {
		newTime := carb.Time.Add(time.Duration(minuteOffset * int64(time.Minute)))
		if newTime.After(time.Now()) {
			return fmt.Errorf("unable to set time after current time")
		}

		_, err = ch.Store.UpdateCarbs(ctx, &defs.Carb{
			ID:     defs.MyObjectID(id),
			Time:   newTime,
			Amount: float64(amount),
		})
		if err != nil {
			return fmt.Errorf("unable to edit carbs: %w", err)
		}
	}

	return ch.updateWithEvent()
}

func (ch *CommandHandler) handleInsulin(data *discord.CommandInteraction) error {
	insulinType := data.Options[0].String()
	units, _ := data.Options[1].FloatValue()

	_, err := ch.Store.WriteInsulin(context.Background(), &defs.Insulin{
		Time:   time.Now().In(ch.Location),
		Amount: units,
		Type:   insulinType,
	})
	if err != nil {
		return fmt.Errorf("unable to save insulin: %w", err)
	}

	return ch.updateWithEvent()
}

func (ch *CommandHandler) handleEditInsulin(data *discord.CommandInteraction) error {
	ctx := context.Background()
	id := data.Options[0].String()

	var ins defs.Insulin
	var err error
	if len(id) == 6 {
		insuls, err := ch.Store.ReadInsulin(
			ctx,
			time.Now().Add(lookbackInterval),
			time.Now(),
		)
		if err != nil {
			return err
		}

		for _, insul := range insuls {
			if hashDigest(string(insul.ID)) == id {
				ins = insul
				break
			}
		}
		if reflect.ValueOf(ins).IsZero() {
			return fmt.Errorf("no entry found with digest %s", id)
		}
	} else {
		err := ch.Store.DocByID(ctx, mg.InsulinCollection, id, &ins)
		if err != nil {
			return err
		}
	}

	var units = ins.Amount
	var insType = ins.Type
	var minuteOffset int64

	for _, opt := range data.Options[1:] {
		switch opt.Name {
		case "units":
			units, err = opt.FloatValue()
		case "type":
			insType = opt.String()
		case "offset":
			minuteOffset, err = opt.IntValue()
		}
		if err != nil {
			return err
		}
	}

	if units < 0 {
		if err := ch.Store.DeleteByID(ctx, mg.InsulinCollection, string(ins.ID)); err != nil {
			return err
		}
	} else {
		newTime := ins.Time.Add(time.Duration(minuteOffset * int64(time.Minute)))
		if newTime.After(time.Now()) {
			return fmt.Errorf("unable to set time after current time")
		}

		_, err = ch.Store.UpdateInsulin(ctx, &defs.Insulin{
			ID:     defs.MyObjectID(id),
			Time:   newTime,
			Amount: units,
			Type:   insType,
		})
		if err != nil {
			return fmt.Errorf("unable to edit insulin: %w", err)
		}
	}

	return ch.updateWithEvent()
}

func (ch *CommandHandler) updateWithEvent() error {
	desc, err := newDescription(ch.Store, ch.Location)
	if err != nil {
		ch.Logger.Debug("unable to generate new description", zap.Error(err))
	}

	oldMessage, err := ch.Display.GetMainMessage()
	if err != nil {
		return err
	}
	oldMessage.Embeds[0].Description = desc

	return ch.Display.UpdateMainMessage(defs.MessageData{
		Embeds: oldMessage.Embeds,
	})
}

func (ch *CommandHandler) handleGenReport(data *discord.CommandInteraction) error {
	timeframe := data.Options[0].String()
	offset, _ := data.Options[1].IntValue()

	start := time.Now().In(ch.Location)
	var end time.Time

	var fr *proto.FileResponse
	var err error

	switch timeframe {
	case "w":
		start = startOfWeek(start)
		start = start.AddDate(0, 0, -int(offset)*7)
		end = start.AddDate(0, 0, 7)

		fr, err = ch.Plotter.GenerateWeeklyPlot(context.Background(), start, end)
		if err != nil {
			ch.Logger.Debug("unable to generate weekly plot", zap.Error(err))
		}
	case "m":
		start = startOfMonth(start.AddDate(0, -int(offset), 0))
		start = start.AddDate(0, 0, 1)
		end = start.AddDate(0, 1, 0)
	}

	fileReader, err := ch.Store.ReadFile(context.Background(), fr.GetId())
	if err != nil {
		ch.Logger.Debug("unable to read file", zap.Error(err))
	}

	if err := ch.Store.DeleteFile(context.Background(), fr.GetId()); err != nil {
		ch.Logger.Debug("unable to delete file", zap.Error(err))
	}

	glucose, err := ch.Store.ReadGlucose(context.Background(), start, end)
	if err != nil {
		return err
	}

	ra := stats.TimeSpentInRange(glucose, ch.GlucoseConfig.Low, ch.GlucoseConfig.High)
	ss := stats.GlucoseSummary(glucose)

	msgData := defs.MessageData{
		Embeds: []defs.EmbedData{
			{
				Title: fmt.Sprintf("%s to %s", start.Format(MonthDayFormat), end.Format(MonthDayFormat)),
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
		ch.Logger.Debug("adding image to embed", zap.String("name", fr.GetName()))
		msgData.Embeds[0].Image = &defs.ImageData{Filename: fr.GetName()}
		msgData.Files = append(msgData.Files, defs.FileData{Name: fr.GetName(), Reader: fileReader})
	}

	_, err = ch.Display.SendMessage(msgData, reportsChannel)
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
