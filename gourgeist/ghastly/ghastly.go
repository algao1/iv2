package ghastly

import (
	"context"
	"fmt"
	"iv2/gourgeist/defs"
	"iv2/gourgeist/ghastly/proto"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Client struct {
	Plotter proto.PlotterClient
	Logger  *zap.Logger
}

type Plotter interface {
	GenerateDailyPlot(context.Context, *PlotData) (*proto.FileResponse, error)
	GenerateWeeklyPlot(ctx context.Context, start, end time.Time) (*proto.FileResponse, error)
}

type PlotData struct {
	Glucose []defs.TransformedReading
	Carbs   []defs.Carb
	Insulin []defs.Insulin
}

func New(conn *grpc.ClientConn, logger *zap.Logger) *Client {
	return &Client{
		Plotter: proto.NewPlotterClient(conn),
		Logger:  logger,
	}
}

func (c *Client) GenerateDailyPlot(ctx context.Context, pd *PlotData) (*proto.FileResponse, error) {
	glucose := make([]*proto.Glucose, len(pd.Glucose))
	for i, g := range pd.Glucose {
		glucose[i] = &proto.Glucose{
			Time:  timestamppb.New(g.Time),
			Value: g.Mmol,
		}
	}

	carbs := make([]*proto.Carb, len(pd.Carbs))
	for i, c := range pd.Carbs {
		carbs[i] = &proto.Carb{
			Time:  timestamppb.New(c.Time),
			Value: c.Amount,
		}
	}

	insulin := make([]*proto.Insulin, len(pd.Insulin))
	for i, ins := range pd.Insulin {
		insulin[i] = &proto.Insulin{
			Time:  timestamppb.New(ins.Time),
			Value: ins.Amount,
		}
	}

	fr, err := c.Plotter.PlotDaily(ctx, &proto.History{
		Glucose: glucose,
		Carbs:   carbs,
		Insulin: insulin,
	})
	if err != nil {
		return nil, fmt.Errorf("unable to plot daily chart: %w", err)
	}

	c.Logger.Debug("successfully obtained plot",
		zap.String("id", fr.GetId()),
		zap.String("name", fr.GetName()),
	)

	return fr, nil
}

func (c *Client) GenerateWeeklyPlot(ctx context.Context, start, end time.Time) (*proto.FileResponse, error) {
	return c.Plotter.PlotWeekly(ctx, &proto.TimeRange{
		Start: timestamppb.New(start),
		End:   timestamppb.New(end),
	})
}
