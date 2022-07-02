package defs

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type TimePoint interface {
	GetTime() time.Time
}

type TransformedReading struct {
	ID    *primitive.ObjectID `bson:"_id,omitempty"`
	Time  time.Time           `bson:"time"`
	Mmol  float64             `bson:"mmol"`
	Trend string              `bson:"trend"`
}

func (tr *TransformedReading) GetTime() time.Time {
	return tr.Time
}

type InsulinType int

const (
	RapidActing InsulinType = iota
	SlowActing
)

func (it InsulinType) String() string {
	return [...]string{"rapid", "slow"}[it]
}

type Insulin struct {
	ID     *primitive.ObjectID `bson:"_id,omitempty"`
	Time   time.Time           `bson:"time"`
	Type   string              `bson:"type"`
	Amount float64             `bson:"amount"`
}

func (in *Insulin) GetTime() time.Time {
	return in.Time
}

type Carb struct {
	ID     *primitive.ObjectID `bson:"_id,omitempty"`
	Time   time.Time           `bson:"time"`
	Amount float64             `bson:"amount"`
}

func (c *Carb) GetTime() time.Time {
	return c.Time
}

type Label int

const (
	HighGlucose Label = iota
	LowGlucose
)

func (l Label) String() string {
	return [...]string{"High Glucose", "Low Glucose"}[l]
}

type Alert struct {
	ID     *primitive.ObjectID `bson:"_id,omitempty"`
	Time   time.Time           `bson:"time"`
	Label  string              `bson:"label"`
	Reason string              `bson:"reason"`
}

func (al *Alert) GetTime() time.Time {
	return al.Time
}
