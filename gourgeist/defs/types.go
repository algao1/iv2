package defs

import (
	"io"
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

// Labels.
const (
	HighGlucoseLabel        = "High Glucose"
	LowGlucoseLabel         = "Low Glucose"
	MissingSlowInsulinLabel = "Missing Slow Acting Insulin"
)

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

// Wrapper types for messages, can eventually be generalized for
// other platforms like Slack.

type MessageData struct {
	Content         string
	Embeds          []EmbedData
	Files           []FileData
	MentionEveryone bool
}

type EmbedData struct {
	Title       string
	Description string
	Fields      []EmbedField
	Image       *ImageData
}

type EmbedField struct {
	Name   string
	Value  string
	Inline bool
}

type ImageData struct {
	Filename string
}

type FileData struct {
	Name   string
	Reader io.Reader
}

func EmptyEmbed() EmbedField {
	return EmbedField{
		Name:   "\u200b",
		Value:  "\u200b",
		Inline: true,
	}
}

// Interactions.

type InteractionResponseType int

const (
	MessageInteraction InteractionResponseType = iota
)

type InteractionResponse struct {
	Type InteractionResponseType
	Data MessageData
}

type CommandInteractionHandler func(EventInfo, CommandInteraction)

type EventInfo struct {
	ID    uint64
	AppID uint64
	Token string
}

type CommandInteraction struct {
	Name    string
	Options []CommandInteractionOption
}

type CommandInteractionOption struct {
	Name  string
	Value string
}
