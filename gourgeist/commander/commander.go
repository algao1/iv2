package commander

import (
	"context"
	"fmt"
	"iv2/gourgeist/defs"
	dcr "iv2/gourgeist/pkg/desc"
	"iv2/gourgeist/pkg/discgo"
	"iv2/gourgeist/pkg/ghastly"
	"iv2/gourgeist/pkg/mg"
	"time"

	"go.uber.org/zap"
)

const monthDayFormat = "01/02"

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
	Descriptor    *dcr.Descriptor
	Location      *time.Location
	GlucoseConfig defs.GlucoseConfig
}

type cleanUp func() error

func (ch *CommandHandler) CreateHandler() func(defs.EventInfo, defs.CommandInteraction) {
	return func(e defs.EventInfo, data defs.CommandInteraction) {
		if err := ch.handleCommand(data); err != nil {
			ch.Logger.Debug("unable to handle command",
				zap.String("command", data.Name),
				zap.Error(err),
			)
		}

		resp := defs.InteractionResponse{
			Type: defs.MessageInteraction,
			Data: defs.MessageData{Content: "received"},
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

func (ch *CommandHandler) handleCommand(data defs.CommandInteraction) error {
	ch.Logger.Debug("received command",
		zap.String("cmd", data.Name),
		zap.Any("options", data.Options),
	)

	switch data.Name {
	case defs.AddCarbsCmd:
		return handleCarbs(ch.Store, data, ch.updateWithEvent)
	case defs.EditCarbsCmd:
		return handleEditCarbs(ch.Store, data, ch.updateWithEvent)
	case defs.AddInsulinCmd:
		return handleInsulin(ch.Store, data, ch.updateWithEvent)
	case defs.EditInsulinCmd:
		return handleEditInsulin(ch.Store, data, ch.updateWithEvent)
	case defs.GenReportCmd:
		// TODO: Handle this better, for now just separate.
		return handleGenReport(
			ch.Store,
			ch.Display,
			ch.Plotter,
			ch.GlucoseConfig,
			ch.Logger,
			ch.Location,
			data,
		)
	case defs.EditVisCmd:
		return handleEditVis(ch.Descriptor, data, ch.updateWithEvent)
	default:
		return fmt.Errorf("unknown command: %s", data.Name)
	}
}

func (ch *CommandHandler) updateWithEvent() error {
	end := time.Now()
	start := end.Add(defs.LookbackInterval)
	ctx := context.Background()

	ins, err := ch.Store.ReadInsulin(ctx, start, end)
	if err != nil {
		return err
	}

	carbs, err := ch.Store.ReadCarbs(ctx, start, end)
	if err != nil {
		return err
	}

	desc, err := ch.Descriptor.New(ins, carbs)
	if err != nil {
		ch.Logger.Debug("unable to generate new dcr", zap.Error(err))
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
