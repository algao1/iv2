import pandas as pd
import plotly.graph_objects as go
import pytz

from datetime import datetime, timedelta
from ghastly.proto.ghastly_pb2 import FileResponse
from ghastly.proto.ghastly_pb2_grpc import PlotterServicer as ps
from loguru import logger
from modules.store import Store
from pytz import timezone

WEEKDAYS = [
    "Sunday",
    "Monday",
    "Tuesday",
    "Wednesday",
    "Thursday",
    "Friday",
    "Saturday",
]


class PlotterServicer(ps):
    def __init__(self, config: dict, store: Store) -> None:
        self.store = store
        self.low = config["glucose"]["low"]
        self.high = config["glucose"]["high"]
        self.target = config["glucose"]["target"]
        self.tz = timezone(config["timezone"])

    def PlotDaily(self, request, context):
        logger.debug("got request to generate daily plot")

        gxs, gys = self.processTimePoints(request.glucose)
        cxs, _ = self.processTimePoints(request.carbs)
        cys = self.interpolateMarker(gxs, gys, cxs, False)
        ixs, _ = self.processTimePoints(request.insulin)
        iys = self.interpolateMarker(gxs, gys, ixs)

        fname = "daily-" + gxs[-1].strftime("%m%d%Y-%H%M%S-%z") + ".png"
        plot = self.plot_daily(gxs, gys, cxs, cys, ixs, iys)
        iid = self.store.store_image(plot, fname)
        return FileResponse(id=f"{iid}", name=fname)

    def PlotWeekly(self, request, context):
        logger.debug("got request to generate weekly plot")

        start, end = request.start, request.end
        start = datetime.fromtimestamp(start.seconds + start.nanos / 1e9).astimezone(
            self.tz
        )
        end = datetime.fromtimestamp(end.seconds + end.nanos / 1e9).astimezone(self.tz)

        glucose = self.store.get_glucose(start, end)

        fname = "weekly-{}.png".format(start.strftime("%m%d"))
        plot = self.plot_weekly(glucose)
        iid = self.store.store_image(plot, fname)
        return FileResponse(id=f"{iid}", name=fname)

    def processTimePoints(self, tps: list):
        # Convert protobuf time to datetime, and localize to timezone.
        xs = [
            datetime.fromtimestamp(tp.time.seconds + tp.time.nanos / 1e9).astimezone(
                self.tz
            )
            for tp in tps
        ]
        ys = [tp.value for tp in tps]
        return xs, ys

    def interpolateMarker(
        self,
        gxs: list[datetime],
        gys: list[float],
        mxs: list[datetime],
        above: bool = True,
    ):
        # Set the proportional offset from the curve.
        res = []
        offset = (max(gys) - min(gys)) / 10
        if not above:
            offset *= -1

        prev_x, prev_y = gxs[0], 0
        i = 0

        for mx in mxs:
            while i < len(gxs) and gxs[i] < mx:
                prev_x, prev_y = gxs[i], gys[i]
                i += 1

            rel_x, rel_y = 0, 0
            if i == 0:
                rel_y = gys[i]
            else:
                # Set the relative height to be proportional to the relative x.
                rel_x = (mx - prev_x) / (gxs[i] - prev_x) if i < len(gxs) else 1
                rel_y = prev_y + rel_x * (gys[i] - prev_y) if i < len(gxs) else gys[-1]
            res.append(rel_y + offset)

        return res

    def defaultLayout(self, fig):
        fig.update_layout(
            width=1400, height=700, margin=dict(l=20, r=20, t=20, b=20),
        )

    def timeseriesLayout(self, fig, x_ll, x_ul, y_ll, y_ul):
        fig.update_layout(
            shapes=[
                dict(  # Draw upper rectangle.
                    type="rect",
                    xref="x",
                    yref="y",
                    x0=x_ll,
                    y0=self.high,
                    x1=x_ul,
                    y1=y_ul,
                    fillcolor="red",
                    opacity=0.15,
                    line_width=0,
                    layer="below",
                ),
                dict(  # Draw lower rectangle.
                    type="rect",
                    xref="x",
                    yref="y",
                    x0=x_ll,
                    y0=y_ll,
                    x1=x_ul,
                    y1=self.low,
                    fillcolor="red",
                    opacity=0.15,
                    line_width=0,
                    layer="below",
                ),
            ],
            xaxis=dict(range=[x_ll, x_ul]),
            yaxis=dict(range=[y_ll, y_ul]),
        )
        fig.add_hline(y=self.low, line_dash="dash", line_color="red")
        fig.add_hline(y=self.high, line_dash="dash", line_color="red")
        fig.add_hline(y=self.target, line_dash="dash", line_color="green")

    def plot_daily(
        self,
        gxs: list[datetime],
        gys: list[float],
        cxs: list[datetime],
        cys: list[float],
        ixs: list[datetime],
        iys: list[float],
    ):
        # Define the limits for bounding boxes.
        x_lowerlim, x_upperlim = (
            gxs[0] + timedelta(minutes=-10),
            gxs[-1] + timedelta(minutes=10),
        )
        y_lowerlim, y_upperlim = 2, max(gys) + 1

        fig = go.Figure()
        fig.add_trace(go.Scatter(name="glucose", x=gxs, y=gys, mode="lines"))
        fig.add_trace(
            go.Scatter(
                name="insulin",
                x=ixs,
                y=iys,
                mode="markers",
                marker_symbol="triangle-down",
                marker_size=15,
            ),
        )
        fig.add_trace(
            go.Scatter(
                name="carbs",
                x=cxs,
                y=cys,
                mode="markers",
                marker_symbol="triangle-up",
                marker_size=15,
            ),
        )

        self.defaultLayout(fig)
        self.timeseriesLayout(fig, x_lowerlim, x_upperlim, y_lowerlim, y_upperlim)

        return fig.to_image(format="png")

    def plot_weekly(self, glucose):
        df = pd.DataFrame(glucose)
        df["time"] = df["time"].dt.tz_localize(pytz.utc)
        df["time"] = df["time"].dt.tz_convert(pytz.timezone("America/Toronto"))

        x_lowerlim = df["time"].iloc[0]
        x_lowerlim += timedelta(days=-x_lowerlim.weekday())
        x_upperlim = x_lowerlim
        y_lowerlim, y_upperlim = 2, 2

        fig = go.Figure()
        for i in range(6):
            day_df = df[df["time"].dt.weekday == i].copy()
            day_df["time"] = day_df["time"] + pd.Timedelta(days=-i)
            fig.add_trace(go.Scatter(name=i, x=day_df["time"], y=day_df["mmol"]))

            # Update limits, not very clean.
            y_upperlim = max(y_upperlim, day_df["mmol"].max())
            x_lowerlim = min(x_lowerlim, day_df["time"].min())
            x_upperlim = max(x_upperlim, day_df["time"].max())

        self.defaultLayout(fig)
        self.timeseriesLayout(fig, x_lowerlim, x_upperlim, y_lowerlim, y_upperlim)

        return fig.to_image(format="png")
