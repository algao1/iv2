package gourgeist

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"iv2/gourgeist/pkg/mg"
	"time"
)

type DescriptionStore interface {
	mg.InsulinStore
	mg.CarbStore
}

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
			idStr := string(ins[i].ID)
			desc += fmt.Sprintf("%s :: (%s) %s\n", ins[i].Time.In(loc).Format(CmdTimeFormat), hashDigest(idStr), idStr)
			desc += fmt.Sprintf("insulin %s %.2f\n", ins[i].Type, ins[i].Amount)
			i--
		} else {
			idStr := string(carbs[j].ID)
			desc += fmt.Sprintf("%s :: (%s) %s\n", carbs[j].Time.In(loc).Format(CmdTimeFormat), hashDigest(idStr), idStr)
			desc += fmt.Sprintf("carbs %.2f\n", carbs[j].Amount)
			j--
		}
	}
	desc += "```"

	return desc, nil
}

func hashDigest(s string) string {
	h := sha1.New()
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))[:6]
}
