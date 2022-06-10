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

type Insulin struct {
	Time   time.Time `bson:"time"`
	Type   string    `bson:"type"`
	Amount float64   `bson:"amount"`
}

func (in *Insulin) GetTime() time.Time {
	return in.Time
}

type Carb struct {
	Time   time.Time `bson:"time"`
	Amount float64   `bson:"amount"`
}

func (c *Carb) GetTime() time.Time {
	return c.Time
}