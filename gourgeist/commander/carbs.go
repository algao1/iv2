package commander

import (
	"context"
	"fmt"
	"iv2/gourgeist/defs"
	"iv2/gourgeist/pkg/mg"
	"strconv"
	"time"
)

func handleCarbs(cs CommanderStore, data defs.CommandInteraction, f cleanUp) error {
	amount, _ := strconv.Atoi(data.Options[0].Value)

	_, err := cs.WriteCarbs(context.Background(), &defs.Carb{
		Time:   time.Now(),
		Amount: float64(amount),
	})
	if err != nil {
		return fmt.Errorf("unable to save carbs: %w", err)
	}

	return f()
}

func handleEditCarbs(cs CommanderStore, data defs.CommandInteraction, f cleanUp) error {
	ctx := context.Background()
	id := data.Options[0].Value

	var carb defs.Carb
	if rec, err := strconv.Atoi(id); err == nil {
		end := time.Now()
		carbs, err := cs.ReadCarbs(ctx, end.Add(defs.LookbackInterval), end)
		if err != nil {
			return err
		}

		if len(carbs) <= rec {
			return err
		}
		id = string(carbs[rec].ID)
	}

	err := cs.DocByID(ctx, mg.CarbsCollection, id, &carb)
	if err != nil {
		return err
	}

	var amount = carb.Amount
	var minuteOffset int

	for _, opt := range data.Options[1:] {
		switch opt.Name {
		case "amount":
			amount, err = strconv.ParseFloat(opt.Value, 64)
		case "offset":
			minuteOffset, err = strconv.Atoi(opt.Value)
		}
		if err != nil {
			return err
		}
	}

	if amount < 0 {
		if err := cs.DeleteByID(ctx, mg.CarbsCollection, string(carb.ID)); err != nil {
			return err
		}
	} else {
		newTime := carb.Time.Add(time.Duration(minuteOffset * int(time.Minute)))
		if newTime.After(time.Now()) {
			return fmt.Errorf("unable to set time after current time")
		}

		_, err = cs.UpdateCarbs(ctx, &defs.Carb{
			ID:     carb.ID,
			Time:   newTime,
			Amount: float64(amount),
		})
		if err != nil {
			return fmt.Errorf("unable to edit carbs: %w", err)
		}
	}

	return f()
}
