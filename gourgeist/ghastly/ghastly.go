package ghastly

import (
	"context"
	"fmt"
	"iv2/gourgeist/ghastly/proto"
	"iv2/gourgeist/types"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Client struct {
	Plotter proto.PlotterClient
	Logger  *zap.Logger
}

type Plotter interface {
	GenerateDailyPlot(ctx context.Context, trs []types.TransformedReading) (*proto.FileResponse, error)
}

func New(conn *grpc.ClientConn, logger *zap.Logger) *Client {
	return &Client{
		Plotter: proto.NewPlotterClient(conn),
		Logger:  logger,
	}
}

func (c *Client) GenerateDailyPlot(ctx context.Context, trs []types.TransformedReading) (*proto.FileResponse, error) {
	glucose := make([]*proto.Glucose, len(trs))
	for i, tr := range trs {
		glucose[i] = &proto.Glucose{
			Time:  timestamppb.New(tr.Time),
			Value: tr.Mmol,
		}
	}

	fr, err := c.Plotter.PlotDaily(ctx, &proto.History{Glucose: glucose})
	if err != nil {
		return nil, fmt.Errorf("unable to plot daily chart: %w", err)
	}

	c.Logger.Debug("successfully obtained plot",
		zap.String("id", fr.GetId()),
		zap.String("name", fr.GetName()),
	)

	return fr, nil
}
