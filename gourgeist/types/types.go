package types

import "time"

type TimePoint interface {
	GetTime() time.Time
}

type TransformedReading struct {
	Time  time.Time `bson:"time"`
	Mmol  float64   `bson:"mmol"`
	Trend string    `bson:"trend"`
}

func (tr *TransformedReading) GetTime() time.Time {
	return tr.Time
}
