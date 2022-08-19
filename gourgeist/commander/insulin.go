package commander

import (
	"context"
	"fmt"
	"iv2/gourgeist/defs"
	dcr "iv2/gourgeist/pkg/desc"
	"iv2/gourgeist/pkg/mg"
	"reflect"
	"strconv"
	"time"
)

func handleInsulin(cs CommanderStore, data defs.CommandInteraction, f cleanUp) error {
	insulinType := data.Options[0].Value
	units, _ := strconv.ParseFloat(data.Options[1].Value, 64)

	_, err := cs.WriteInsulin(context.Background(), &defs.Insulin{
		Time:   time.Now(),
		Amount: units,
		Type:   insulinType,
	})
	if err != nil {
		return fmt.Errorf("unable to save insulin: %w", err)
	}

	return f()
}

func handleEditInsulin(cs CommanderStore, data defs.CommandInteraction, f cleanUp) error {
	ctx := context.Background()
	id := data.Options[0].Value

	var ins defs.Insulin
	var err error
	if len(id) == 6 {
		insuls, err := cs.ReadInsulin(
			ctx,
			time.Now().Add(defs.LookbackInterval),
			time.Now(),
		)
		if err != nil {
			return err
		}

		for _, insul := range insuls {
			if dcr.HashDigest(string(insul.ID)) == id {
				ins = insul
				break
			}
		}
		if reflect.ValueOf(ins).IsZero() {
			return fmt.Errorf("no entry found with digest %s", id)
		}
	} else {
		err := cs.DocByID(ctx, mg.InsulinCollection, id, &ins)
		if err != nil {
			return err
		}
	}

	var units = ins.Amount
	var insType = ins.Type
	var minuteOffset int

	for _, opt := range data.Options[1:] {
		switch opt.Name {
		case "units":
			units, err = strconv.ParseFloat(opt.Value, 64)
		case "type":
			insType = opt.Value
		case "offset":
			minuteOffset, err = strconv.Atoi(opt.Value)
		}
		if err != nil {
			return err
		}
	}

	if units < 0 {
		if err := cs.DeleteByID(ctx, mg.InsulinCollection, string(ins.ID)); err != nil {
			return err
		}
	} else {
		newTime := ins.Time.Add(time.Duration(minuteOffset * int(time.Minute)))
		if newTime.After(time.Now()) {
			return fmt.Errorf("unable to set time after current time")
		}

		_, err = cs.UpdateInsulin(ctx, &defs.Insulin{
			ID:     ins.ID,
			Time:   newTime,
			Amount: units,
			Type:   insType,
		})
		if err != nil {
			return fmt.Errorf("unable to edit insulin: %w", err)
		}
	}

	return f()
}
