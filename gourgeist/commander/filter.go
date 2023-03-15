package commander

import (
	"iv2/gourgeist/defs"
	dcr "iv2/gourgeist/pkg/desc"
	"strconv"
)

func handleEditVis(d *dcr.Descriptor, data defs.CommandInteraction, f cleanUp) error {
	visStr := data.Options[0].Value
	vis, err := strconv.Atoi(visStr)
	if err != nil {
		return err
	}
	d.Set(defs.Visibility(vis))

	return f()
}
