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
		var err error
		switch data := e.Data.(type) {
		case *discord.CommandInteraction:
			err = ch.handleCommand(data)
		}

		if err != nil {
			ch.Logger.Debug("unable to handle command", zap.Error(err))
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
	ch.Logger.Debug("received command", zap.String("cmd", data.Name))

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
		return fmt.Errorf("received unknown command: %s", data.Name)
	}
}

func (ch *CommandHandler) handleCarbs(data *discord.CommandInteraction) error {
	amount, _ := data.Options[0].IntValue()
	ch.Logger.Debug("carbs", zap.Int("amount", int(amount)))

	_, err := ch.Store.WriteCarbs(context.Background(), &defs.Carb{
		Time:   time.Now().In(ch.Location),
		Amount: float64(amount),
	})
	if err != nil {
		return fmt.Errorf("unable to save carbs: %w", err)
	}

	err = ch.updateWithEvent()
	if err != nil {
		return fmt.Errorf("unable to complete carbs command: %w", err)
	}

	return nil
}

func (ch *CommandHandler) handleEditCarbs(data *discord.CommandInteraction) error {
	id := data.Options[0].String()

	var carbs defs.Carb
	err := ch.Store.DocByID(context.Background(), mg.CarbsCollection, id, &carbs)
	if err != nil {
		return err
	}

	var amount = carbs.Amount
	var minuteOffset int64

	for _, opt := range data.Options[1:] {
		switch opt.Name {
		case "amount":
			amount, err = opt.FloatValue()
		case "offset":
			minuteOffset, err = opt.IntValue()
		}
	}
	if err != nil {
		return err
	}

	if amount < 0 {
		if err := ch.Store.DeleteByID(context.Background(), mg.CarbsCollection, id); err != nil {
			return err
		}
	} else {
		newTime := carbs.Time.Add(time.Duration(minuteOffset * int64(time.Minute)))
		if newTime.After(time.Now()) {
			return fmt.Errorf("unable to set time after current time")
		}

		_, err = ch.Store.UpdateCarbs(context.Background(), &defs.Carb{
			ID:     defs.MyObjectID(id),
			Time:   newTime,
			Amount: float64(amount),
		})
		if err != nil {
			return fmt.Errorf("unable to edit carbs: %w", err)
		}
	}

	err = ch.updateWithEvent()
	if err != nil {
		return fmt.Errorf("unable to complete editcarbs command: %w", err)
	}

	return nil
}

func (ch *CommandHandler) handleInsulin(data *discord.CommandInteraction) error {
	insulinType := data.Options[0].String()
	units, _ := data.Options[1].FloatValue()
	ch.Logger.Debug("insulin", zap.Float64("units", units), zap.String("type", insulinType))

	_, err := ch.Store.WriteInsulin(context.Background(), &defs.Insulin{
		Time:   time.Now().In(ch.Location),
		Amount: units,
		Type:   insulinType,
	})
	if err != nil {
		return fmt.Errorf("unable to save insulin: %w", err)
	}

	err = ch.updateWithEvent()
	if err != nil {
		return fmt.Errorf("unable to complete insulin command: %w", err)
	}

	return nil
}

func (ch *CommandHandler) handleEditInsulin(data *discord.CommandInteraction) error {
	id := data.Options[0].String()

	var ins defs.Insulin
	err := ch.Store.DocByID(context.Background(), mg.InsulinCollection, id, &ins)
	if err != nil {
		return err
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
	}
	if err != nil {
		return err
	}

	if units < 0 {
		if err := ch.Store.DeleteByID(context.Background(), mg.InsulinCollection, id); err != nil {
			return err
		}
	} else {
		newTime := ins.Time.Add(time.Duration(minuteOffset * int64(time.Minute)))
		if newTime.After(time.Now()) {
			return fmt.Errorf("unable to set time after current time")
		}

		_, err = ch.Store.UpdateInsulin(context.Background(), &defs.Insulin{
			ID:     defs.MyObjectID(id),
			Time:   newTime,
			Amount: units,
			Type:   insType,
		})
		if err != nil {
			return fmt.Errorf("unable to edit insulin: %w", err)
		}
	}

	err = ch.updateWithEvent()
	if err != nil {
		return fmt.Errorf("unable to complete editinsulin command: %w", err)
	}

	return nil
}

func (ch *CommandHandler) updateWithEvent() error {
	desc, err := newDescription(ch.Store, ch.Location)
	if err != nil {
		ch.Logger.Debug("unable to generate new description", zap.Error(err))
	}

	oldMessage, err := ch.Display.GetMainMessage()
	if err != nil {
		return fmt.Errorf("unable to complete editcarbs command: %w", err)
	}
	ch.Logger.Debug("old message", zap.Any("embeds", oldMessage.Embeds))

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

type DescriptionStore interface {
	mg.InsulinStore
	mg.CarbStore
}

// TODO: Shouldn't really be put here, need to relocate.

func newDescription(s DescriptionStore, loc *time.Location) (string, error) {
	end := time.Now().In(loc)
	start := end.Add(lookbackInterval)

	ins, err := s.ReadInsulin(context.Background(), start, end)
	if err != nil {
		return "", err
	}

	carbs, err := s.ReadCarbs(context.Background(), start, end)
	if err != nil {
		return "", err
	}

	max_len := logLimit
	if len(ins)+len(carbs) < max_len {
		max_len = len(ins) + len(carbs)
	}
	if max_len == 0 {
		return "", nil
	}

	desc := "```"
	i := len(ins) - 1
	j := len(carbs) - 1
	for t := 0; t < max_len; t++ {
		// TODO: Make this cleaner.
		if i >= 0 && (j < 0 || ins[i].Time.After(carbs[j].Time)) {
			desc += fmt.Sprintf("%s :: %s\n",
				ins[i].Time.In(loc).Format(CmdTimeFormat),
				ins[i].ID,
			)
			desc += fmt.Sprintf("insulin %s %.2f\n", ins[i].Type, ins[i].Amount)
			i--
		} else {
			desc += fmt.Sprintf("%s :: %s\n",
				carbs[j].Time.In(loc).Format(CmdTimeFormat),
				carbs[j].ID,
			)
			desc += fmt.Sprintf("carbs %.2f\n", carbs[j].Amount)
			j--
		}
	}
	desc += "```"

	return desc, nil
}
