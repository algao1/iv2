package types

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

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

type CommandEvent struct {
	Time      time.Time `bson:"time"`
	CmdString string    `bson:"cmdString"`
}

func (cmd *CommandEvent) GetTime() time.Time {
	return cmd.Time
}
