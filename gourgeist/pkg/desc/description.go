package desc

import (
	"fmt"
	"iv2/gourgeist/defs"
	"sort"
	"sync"
	"time"
)

const (
	logLimit      = 7
	cmdTimeFormat = "03:04 PM"
)

type Descriptor struct {
	Vis defs.Visibility
	Loc *time.Location

	sync.Mutex
}

func New(loc *time.Location) *Descriptor {
	return &Descriptor{Loc: loc}
}

func (d *Descriptor) Wrap(desc string) string {
	return "```" + desc + "```"
}

func (d *Descriptor) Set(vis defs.Visibility) {
	d.Mutex.Lock()
	defer d.Mutex.Unlock()
	d.Vis = vis
}

func (d *Descriptor) New(ins []defs.Insulin, carbs []defs.Carb) (string, error) {
	d.Mutex.Lock()
	defer d.Mutex.Unlock()

	entries := make([]struct {
		time.Time
		string
	}, 0)

	if d.Vis == defs.CarbInsulin || d.Vis == defs.InsulinOnly {
		for _, in := range ins {
			entries = append(entries, struct {
				time.Time
				string
			}{in.Time, fmt.Sprintf("insulin %s %.2f", in.Type, in.Amount)})
		}
	}

	if d.Vis == defs.CarbInsulin || d.Vis == defs.CarbOnly {
		for _, c := range carbs {
			entries = append(entries, struct {
				time.Time
				string
			}{c.Time, fmt.Sprintf("carbs %.2f", c.Amount)})
		}
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Time.After(entries[j].Time)
	})
	if len(entries) > logLimit {
		entries = entries[:logLimit]
	}

	var desc string
	for i := range entries {
		desc += fmt.Sprintf("[%d] %s :: ",
			i, entries[i].Time.In(d.Loc).Format(cmdTimeFormat),
		)
		desc += entries[i].string + "\n"
	}

	if len(desc) > 0 {
		switch d.Vis {
		case defs.CarbOnly:
			desc = "Carb \n" + desc
		case defs.InsulinOnly:
			desc = "Insulin \n" + desc
		default:
			desc = "All \n" + desc
		}
		desc = d.Wrap(desc)
	}

	return desc, nil
}
