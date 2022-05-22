package ghastly

import (
	"context"
	"iv2/gourgeist/dexcom"
	"iv2/gourgeist/ghastly/proto"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Client struct {
	Plotter proto.PlotterClient
	Logger  *zap.Logger
}

type Plotter interface {
	GenerateDailyPlot(ctx context.Context, trs []dexcom.TransformedReading) (*proto.FileResponse, error)
}

func New(conn *grpc.ClientConn, logger *zap.Logger) *Client {
	return &Client{
		Plotter: proto.NewPlotterClient(conn),
		Logger:  logger,
	}
}

func (c *Client) GenerateDailyPlot(ctx context.Context, trs []dexcom.TransformedReading) (*proto.FileResponse, error) {
	tps := make([]*proto.TimePoint, len(trs))
	for i, tr := range trs {
		tps[i] = &proto.TimePoint{
			Time:  timestamppb.New(tr.Time),
			Value: tr.Mmol,
		}
	}

	fr, err := c.Plotter.PlotDaily(ctx, &proto.History{Tps: tps})
	if err != nil {
		return nil, err
	}

	c.Logger.Debug("successfully obtained plot",
		zap.String("id", fr.GetId()),
		zap.String("name", fr.GetName()),
	)

	return fr, nil
}
