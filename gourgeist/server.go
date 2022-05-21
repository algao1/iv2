package gourgeist

import (
	"context"
	"iv2/gourgeist/dexcom"
	"iv2/gourgeist/discgo"
	"iv2/gourgeist/ghastly"
	"iv2/gourgeist/store"
	"strconv"
	"time"

	"github.com/diamondburned/arikawa/api"
	"github.com/diamondburned/arikawa/discord"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	DownloaderInterval = 1 * time.Minute
	UpdaterInterval    = DownloaderInterval

	timeoutInterval  = 2 * time.Second
	lookbackInterval = -12 * time.Hour

	defaultDBName = "ichor"
)

type Server struct {
	Dexcom   *dexcom.Client
	Discord  *discgo.Discord
	Ghastly  *ghastly.Client
	Store    store.Store
	Logger   *zap.Logger
	Location *time.Location
}

type Config struct {
	DexcomAccount  string `yaml:"dexcomAccount"`
	DexcomPassword string `yaml:"dexcomPassword"`
	DiscordToken   string `yaml:"discordToken"`
	DiscordGuild   string `yaml:"discordGuild"`
	MongoURI       string `yaml:"mongoURI"`
	TrevenantAddr  string `yaml:"trevenantAddress"`
	Timezone       string `yaml:"timezone"`
	Logger         *zap.Logger
}

func New(config Config) (*Server, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeoutInterval)
	defer cancel()

	var err error

	loc := time.Local
	if config.Timezone != "" {
		loc, err = time.LoadLocation(config.Timezone)
		if err != nil {
			return nil, err
		}
	}

	ms, err := store.New(ctx, config.MongoURI, defaultDBName, config.Logger)
	if err != nil {
		return nil, err
	}

	dexcom := dexcom.New(config.DexcomAccount, config.DexcomPassword, config.Logger)

	discgo, err := discgo.New(config.DiscordToken, config.Logger, loc)
	if err != nil {
		return nil, err
	}
	err = discgo.Setup(config.DiscordGuild)
	if err != nil {
		return nil, err
	}

	conn, err := grpc.Dial(config.TrevenantAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	gh := ghastly.New(conn, config.Logger)

	config.Logger.Debug("finished server setup", zap.Any("config", config))

	return &Server{
		Dexcom:   dexcom,
		Discord:  discgo,
		Ghastly:  gh,
		Store:    ms,
		Logger:   config.Logger,
		Location: loc,
	}, nil
}

// TODO: Functions below need to be updated/refactored.

func (s *Server) ExecuteTask(interval time.Duration, task func()) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for ; true; <-ticker.C {
		task()
	}
}

func (s *Server) UpdateDiscord() {
	now := time.Now().UTC()
	trs, err := s.Store.ReadGlucose(context.Background(), now.Add(lookbackInterval), now)
	if err != nil {
		s.Logger.Debug("unable to read glucose from store", zap.Error(err))
		return
	}

	if len(trs) == 0 {
		s.Logger.Debug("no glucose readings found")
		return
	}

	fr, err := s.Ghastly.GenerateDailyPlot(context.Background(), trs)
	if err != nil {
		s.Logger.Debug("unable to generate daily plot", zap.Error(err))
	}

	if fr.GetId() == "-1" {
		s.Logger.Debug("unable to generate daily plot")
	}

	fileReader, err := s.Store.ReadFile(context.Background(), fr.GetId())
	if err != nil {
		s.Logger.Debug("unable to read file", zap.Error(err))
	}

	if err := s.Store.DeleteFile(context.Background(), fr.GetId()); err != nil {
		s.Logger.Debug("unable to delete file", zap.Error(err))
	}

	tr := trs[len(trs)-1]
	embed := discord.Embed{
		Title: tr.Time.In(s.Location).Format(discgo.TimeFormat),
		Fields: []discord.EmbedField{
			{Name: "Current", Value: strconv.FormatFloat(tr.Mmol, 'f', 2, 64)},
		},
	}
	msgData := api.SendMessageData{
		Embed: &embed,
		Files: []api.SendMessageFile{},
	}

	if fileReader != nil {
		s.Logger.Debug("adding image to embed", zap.String("name", fr.GetName()))
		embed.Image = &discord.EmbedImage{URL: "attachment://" + fr.GetName()}
		msgData.Files = append(msgData.Files, api.SendMessageFile{Name: fr.GetName(), Reader: fileReader})
	}

	s.Discord.UpdateMainMessage(msgData)
}

func (s *Server) FetchUploadReadings() {
	trs, _ := s.Dexcom.Readings(context.Background(), dexcom.MinuteLimit, dexcom.CountLimit)
	for _, tr := range trs {
		exist, err := s.Store.WriteGlucose(context.Background(), tr)
		if err != nil {
			s.Logger.Debug("unable to write glucose to store", zap.Error(err))
		}
		if exist {
			return
		}
	}
}
