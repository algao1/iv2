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

        start, end = request.start, request.end
        start = datetime.fromtimestamp(start.seconds + start.nanos / 1e9).astimezone(
            self.tz
        )
        end = datetime.fromtimestamp(end.seconds + end.nanos / 1e9).astimezone(self.tz)

        glucose = self.store.get_glucose(start, end)
        carbs = self.store.get_carbs(start, end)
        insulin = self.store.get_insulin(start, end)

        # Process and interpolate points.
        gxs = self.process_timepoints(glucose)
        cxs = self.process_timepoints(carbs)
        ixs = self.process_timepoints(insulin)

        gys = [g["mmol"] for g in glucose]
        cys = self.interpolate_markers(gxs, gys, cxs, False)
        iys = self.interpolate_markers(gxs, gys, ixs)

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
        y_lowerlim, y_upperlim = 2, max(gys)

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

        self.default_layout(fig)
        self.timeseries_layout(fig, x_lowerlim, x_upperlim, y_lowerlim, y_upperlim)

        return fig.to_image(format="png")

    def plot_weekly(self, glucose):
        df = pd.DataFrame(glucose)
        df["time"] = df["time"].dt.tz_localize(pytz.utc)  # type: ignore
        df["time"] = df["time"].dt.tz_convert(pytz.timezone("America/Toronto"))  # type: ignore

        x_lowerlim = df["time"].iloc[0]
        x_lowerlim += timedelta(days=-x_lowerlim.weekday())
        x_upperlim = x_lowerlim
        y_lowerlim, y_upperlim = 2, 2

        fig = go.Figure()
        for i in range(6):
            day_df = df[df["time"].dt.weekday == i].copy()  # type: ignore
            day_df["time"] = day_df["time"] + pd.Timedelta(days=-i)
            fig.add_trace(
                go.Scatter(name=WEEKDAYS[i], x=day_df["time"], y=day_df["mmol"])
            )

            # Update limits, not very clean.
            y_upperlim = max(y_upperlim, day_df["mmol"].max())
            x_lowerlim = min(x_lowerlim, day_df["time"].min())
            x_upperlim = max(x_upperlim, day_df["time"].max())

        self.default_layout(fig)
        self.timeseries_layout(fig, x_lowerlim, x_upperlim, y_lowerlim, y_upperlim)

        return fig.to_image(format="png")

    def process_timepoints(self, tps: list):
        # Localize timezone as UTC and convert to local timezone.
        return [pytz.utc.localize(t["time"]).astimezone(self.tz) for t in tps]

    def interpolate_markers(
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

    def default_layout(self, fig):
        fig.update_layout(
            width=1400, height=700, margin=dict(l=20, r=20, t=20, b=20),
        )

    def timeseries_layout(self, fig, x_ll, x_ul, y_ll, y_ul):
        fig.update_layout(
            shapes=[
                dict(  # Draw upper rectangle.
                    type="rect",
                    xref="x",
                    yref="y",
                    x0=x_ll,
                    y0=self.high,
                    x1=x_ul,
                    y1=y_ul + 2,
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
                dict(  # Draw target region.
                    type="rect",
                    xref="x",
                    yref="y",
                    x0=x_ll,
                    y0=self.target - 1,
                    x1=x_ul,
                    y1=self.target + 1,
                    fillcolor="green",
                    opacity=0.15,
                    line_width=0,
                    layer="below",
                ),
            ],
            xaxis=dict(range=[x_ll, x_ul]),
            yaxis=dict(range=[y_ll, y_ul + 2]),
        )
        fig.add_hline(y=self.low, line_dash="dash", line_color="red")
        fig.add_hline(y=self.high, line_dash="dash", line_color="red")
        fig.add_hline(y=self.target, line_dash="dash", line_color="green")
