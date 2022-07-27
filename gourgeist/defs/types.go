package defs

import (
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type MyObjectID string

func (id MyObjectID) MarshalBSONValue() (bsontype.Type, []byte, error) {
	p, err := primitive.ObjectIDFromHex(string(id))
	if err != nil {
		return bsontype.Null, nil, err
	}
	return bson.MarshalValue(p)
}

type TransformedReading struct {
	ID    MyObjectID `bson:"_id,omitempty"`
	Time  time.Time  `bson:"time"`
	Mmol  float64    `bson:"mmol"`
	Trend string     `bson:"trend"`
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
	ID     MyObjectID `bson:"_id,omitempty"`
	Time   time.Time  `bson:"time"`
	Type   string     `bson:"type"`
	Amount float64    `bson:"amount"`
}

type Carb struct {
	ID     MyObjectID `bson:"_id,omitempty"`
	Time   time.Time  `bson:"time"`
	Amount float64    `bson:"amount"`
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
	ID     MyObjectID `bson:"_id,omitempty"`
	Time   time.Time  `bson:"time"`
	Label  string     `bson:"label"`
	Reason string     `bson:"reason"`
}

// Wrapper for mongo.UpdateResult.
type UpdateResult struct {
	MatchedCount  int64
	ModifiedCount int64
	UpsertedCount int64
	UpsertedID    MyObjectID
}
