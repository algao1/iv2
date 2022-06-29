package gourgeist

import (
	"context"
	"fmt"
	"iv2/gourgeist/discgo"
	"iv2/gourgeist/mg"
	"iv2/gourgeist/types"
	"time"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/utils/json/option"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
)

const (
	CmdTimeFormat  = "03:04 PM"
	ExpireDuration = 8 * time.Hour
	LogLimit       = 5
)

type CommandHandler struct {
	Display discgo.Display
	Store   mg.Store

	Logger   *zap.Logger
	Location *time.Location
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
			return
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
			ch.Logger.Debug("unable to delete interaction response", zap.Error(err))
		}
	}
}

func (ch *CommandHandler) handleCommand(data *discord.CommandInteraction) error {
	ch.Logger.Debug("received command", zap.String("cmd", data.Name))

	switch data.Name {
	case discgo.AddCarbsCmd:
		return ch.handleCarbs(data)
	case discgo.EditCarbsCmd:
		return ch.handleEditCarbs(data)
	case discgo.AddInsulinCmd:
		return ch.handleInsulin(data)
	case discgo.EditInsulinCmd:
		return ch.handleEditInsulin(data)
	default:
		return fmt.Errorf("received unknown command: %s", data.Name)
	}
}

func (ch *CommandHandler) handleCarbs(data *discord.CommandInteraction) error {
	amount, _ := data.Options[0].IntValue()
	ch.Logger.Debug("carbs", zap.Int("amount", int(amount)))

	oldMessage, err := ch.Display.GetMainMessage()
	if err != nil {
		return fmt.Errorf("unable to complete carbs command: %w", err)
	}
	ch.Logger.Debug("old message", zap.Any("embeds", oldMessage.Embeds))

	_, err = ch.Store.WriteCarbs(context.Background(), &types.Carb{
		Time:   time.Now().In(ch.Location),
		Amount: float64(amount),
	})
	if err != nil {
		return fmt.Errorf("unable to save carbs: %w", err)
	}

	err = ch.updateWithEvent(oldMessage)
	if err != nil {
		return fmt.Errorf("unable to complete carbs command: %w", err)
	}

	return nil
}

func (ch *CommandHandler) handleEditCarbs(data *discord.CommandInteraction) error {
	id := data.Options[0].String()
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}
	amount, _ := data.Options[1].FloatValue()

	oldMessage, err := ch.Display.GetMainMessage()
	if err != nil {
		return fmt.Errorf("unable to complete editcarbs command: %w", err)
	}
	ch.Logger.Debug("old message", zap.Any("embeds", oldMessage.Embeds))

	var carbs types.Carb
	if amount < 0 {
		if err := ch.Store.DeleteByID(context.Background(), mg.CarbsCollection, &oid); err != nil {
			return err
		}
	} else {
		if err := ch.Store.DocByID(context.Background(), mg.CarbsCollection, &oid, &carbs); err != nil {
			return err
		}
		_, err = ch.Store.WriteCarbs(context.Background(), &types.Carb{
			Time:   carbs.Time,
			Amount: float64(amount),
		})
		if err != nil {
			return fmt.Errorf("unable to edit carbs: %w", err)
		}
	}

	err = ch.updateWithEvent(oldMessage)
	if err != nil {
		return fmt.Errorf("unable to complete editcarbs command: %w", err)
	}

	return nil
}

func (ch *CommandHandler) handleInsulin(data *discord.CommandInteraction) error {
	insulinType := data.Options[0].String()
	units, _ := data.Options[1].FloatValue()
	ch.Logger.Debug("insulin", zap.Float64("units", units), zap.String("type", insulinType))

	oldMessage, err := ch.Display.GetMainMessage()
	if err != nil {
		return fmt.Errorf("unable to complete insulin command: %w", err)
	}
	ch.Logger.Debug("old message", zap.Any("embeds", oldMessage.Embeds))

	_, err = ch.Store.WriteInsulin(context.Background(), &types.Insulin{
		Time:   time.Now().In(ch.Location),
		Amount: units,
		Type:   insulinType,
	})
	if err != nil {
		return fmt.Errorf("unable to save insulin: %w", err)
	}

	err = ch.updateWithEvent(oldMessage)
	if err != nil {
		return fmt.Errorf("unable to complete insulin command: %w", err)
	}

	return nil
}

func (ch *CommandHandler) handleEditInsulin(data *discord.CommandInteraction) error {
	id := data.Options[0].String()
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	var ins types.Insulin
	if err := ch.Store.DocByID(context.Background(), mg.InsulinCollection, &oid, &ins); err != nil {
		return err
	}

	var units = ins.Amount
	var itype = ins.Type

	for _, opt := range data.Options[1:] {
		switch opt.Name {
		case "units":
			units, err = opt.FloatValue()
		case "type":
			itype = opt.String()
		}
	}
	if err != nil {
		return err
	}

	oldMessage, err := ch.Display.GetMainMessage()
	if err != nil {
		return fmt.Errorf("unable to complete editcarbs command: %w", err)
	}
	ch.Logger.Debug("old message", zap.Any("embeds", oldMessage.Embeds))

	if units < 0 {
		if err := ch.Store.DeleteByID(context.Background(), mg.InsulinCollection, &oid); err != nil {
			return err
		}
	} else {
		_, err = ch.Store.WriteInsulin(context.Background(), &types.Insulin{
			Time:   ins.Time,
			Amount: units,
			Type:   itype,
		})
		if err != nil {
			return fmt.Errorf("unable to edit insulin: %w", err)
		}
	}

	err = ch.updateWithEvent(oldMessage)
	if err != nil {
		return fmt.Errorf("unable to complete editinsulin command: %w", err)
	}

	return nil
}

func (ch *CommandHandler) updateWithEvent(oldMessage *discord.Message) error {
	desc, err := newDescription(ch.Store, ch.Location)
	if err != nil {
		ch.Logger.Debug("unable to generate new description", zap.Error(err))
	}
	oldMessage.Embeds[0].Description = desc

	return ch.Display.UpdateMainMessage(api.EditMessageData{
		Embeds:      &oldMessage.Embeds,
		Attachments: &[]discord.Attachment{},
	})
}

func newDescription(s mg.Store, loc *time.Location) (string, error) {
	end := time.Now().In(loc)
	start := end.Add(-ExpireDuration)

	ins, err := s.ReadInsulin(context.Background(), start, end)
	if err != nil {
		return "", err
	}

	carbs, err := s.ReadCarbs(context.Background(), start, end)
	if err != nil {
		return "", err
	}

	max_len := LogLimit
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
		if i >= 0 {
			desc += fmt.Sprintf("%s :: %s\n",
				ins[i].Time.In(loc).Format(CmdTimeFormat),
				ins[i].ID.Hex(),
			)
			desc += fmt.Sprintf("insulin %.2f %s\n", ins[i].Amount, ins[i].Type)
			i--
		} else {
			desc += fmt.Sprintf("%s :: %s\n",
				carbs[j].Time.In(loc).Format(CmdTimeFormat),
				carbs[j].ID.Hex(),
			)
			desc += fmt.Sprintf("carbs %.2f\n", carbs[j].Amount)
			j--
		}
	}
	desc += "```"

	return desc, nil
}
