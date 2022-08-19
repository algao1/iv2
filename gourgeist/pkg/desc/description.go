package desc

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"iv2/gourgeist/defs"
	"time"
)

const (
	logLimit      = 7
	cmdTimeFormat = "03:04 PM"
)

func Wrap(desc string) string {
	return "```" + desc + "```"
}

func New(ins []defs.Insulin, carbs []defs.Carb, loc *time.Location) (string, error) {
	max_len := logLimit
	if len(ins)+len(carbs) < max_len {
		max_len = len(ins) + len(carbs)
	}
	if max_len == 0 {
		return "", nil
	}

	var desc string
	i := len(ins) - 1
	j := len(carbs) - 1
	for t := 0; t < max_len; t++ {
		// TODO: Make this cleaner.
		if i >= 0 && (j < 0 || ins[i].Time.After(carbs[j].Time)) {
			idStr := string(ins[i].ID)
			desc += fmt.Sprintf("%s :: (%s) %s\n",
				ins[i].Time.In(loc).Format(cmdTimeFormat), HashDigest(idStr), idStr)
			desc += fmt.Sprintf("insulin %s %.2f\n", ins[i].Type, ins[i].Amount)
			i--
		} else {
			idStr := string(carbs[j].ID)
			desc += fmt.Sprintf("%s :: (%s) %s\n",
				carbs[j].Time.In(loc).Format(cmdTimeFormat), HashDigest(idStr), idStr)
			desc += fmt.Sprintf("carbs %.2f\n", carbs[j].Amount)
			j--
		}
	}

	return Wrap(desc), nil
}

func HashDigest(s string) string {
	h := sha1.New()
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))[:6]
}
