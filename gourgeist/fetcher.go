package gourgeist

import (
	"context"
	"fmt"
	"iv2/gourgeist/dexcom"
	"iv2/gourgeist/mg"

	"go.uber.org/zap"
)

type FetcherStore interface {
	mg.GlucoseStore
}

type Fetcher struct {
	Source dexcom.Source
	Store  FetcherStore

	Logger *zap.Logger
}

func (f *Fetcher) FetchAndLoad() error {
	trs, _ := f.Source.Readings(context.Background(), dexcom.MinuteLimit, dexcom.CountLimit)
	for _, tr := range trs {
		res, err := f.Store.WriteGlucose(context.Background(), tr)
		if err != nil {
			return fmt.Errorf("unable to write glucose to store: %w", err)
		}
		if res.MatchedCount > 0 { // Matched.
			break
		}
	}
	return nil
}
