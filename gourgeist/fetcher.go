package gourgeist

import (
	"context"
	"fmt"
	"iv2/gourgeist/dexcom"
	"iv2/gourgeist/mongo"

	"go.uber.org/zap"
)

type Fetcher struct {
	Source dexcom.Source
	Store  mongo.Store

	Logger *zap.Logger
}

func (f *Fetcher) FetchAndLoad() error {
	trs, _ := f.Source.Readings(context.Background(), dexcom.MinuteLimit, dexcom.CountLimit)
	for _, tr := range trs {
		exist, err := f.Store.WriteGlucose(context.Background(), tr)
		if err != nil {
			return fmt.Errorf("unable to write glucose to store: %w", err)
		}
		if exist {
			break
		}
	}
	return nil
}
