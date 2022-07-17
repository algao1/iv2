package ghastly

import (
	"context"
	"iv2/gourgeist/pkg/ghastly/proto"
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
	GenerateDailyPlot(ctx context.Context, start, end time.Time) (*proto.FileResponse, error)
	GenerateWeeklyPlot(ctx context.Context, start, end time.Time) (*proto.FileResponse, error)
}

func New(conn *grpc.ClientConn, logger *zap.Logger) *Client {
	return &Client{
		Plotter: proto.NewPlotterClient(conn),
		Logger:  logger,
	}
}

func (c *Client) GenerateDailyPlot(ctx context.Context, start, end time.Time) (*proto.FileResponse, error) {
	return c.Plotter.PlotDaily(ctx, &proto.TimeRange{
		Start: timestamppb.New(start),
		End:   timestamppb.New(end),
	})
}

func (c *Client) GenerateWeeklyPlot(ctx context.Context, start, end time.Time) (*proto.FileResponse, error) {
	return c.Plotter.PlotWeekly(ctx, &proto.TimeRange{
		Start: timestamppb.New(start),
		End:   timestamppb.New(end),
	})
}
