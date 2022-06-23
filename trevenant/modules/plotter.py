import gc
import plotly.graph_objects as go

from datetime import datetime
from ghastly.proto.ghastly_pb2 import FileResponse
from ghastly.proto.ghastly_pb2_grpc import PlotterServicer as ps
from loguru import logger
from pytz import timezone

from modules.store import Store

# TODO:
# - add unit tests
# - add type hints


class PlotterServicer(ps):
    def __init__(self, store: Store) -> None:
        self.store = store
        self.tz = timezone("US/Eastern")

    def PlotDaily(self, request, context):
        logger.debug("got request to generate daily plot")

        gxs, gys = self.processTimePoints(request.glucose)
        cxs, _ = self.processTimePoints(request.carbs)
        cys = self.interpolateMarker(gxs, gys, cxs, False)
        ixs, _ = self.processTimePoints(request.insulin)
        iys = self.interpolateMarker(gxs, gys, ixs)

        fname = "daily-" + gxs[-1].strftime("%m%d%Y-%H%M%S-%z") + ".png"
        iid = self.store.store_image(self.plot(gxs, gys, cxs, cys, ixs, iys), fname)
        return FileResponse(id=f"{iid}", name=fname)

    def processTimePoints(self, tps: list):
        xs = [
            self.tz.localize(
                datetime.fromtimestamp(tp.time.seconds + tp.time.nanos / 1e9)
            )
            for tp in tps
        ]
        ys = [tp.value for tp in tps]
        return xs, ys

    # TODO: Make this better.
    def interpolateMarker(
        self,
        gxs: list[datetime],
        gys: list[float],
        mxs: list[datetime],
        above: bool = True,
    ):
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
                rel_x = (mx - prev_x) / (gxs[i] - prev_x) if i < len(gxs) else 1
                rel_y = prev_y + rel_x * (gys[i] - prev_y) if i < len(gxs) else gys[-1]
            res.append(rel_y + offset)

        return res

    def plot(
        self,
        gxs: list[datetime],
        gys: list[float],
        cxs: list[datetime],
        cys: list[float],
        ixs: list[datetime],
        iys: list[float],
    ):
        y_lim = max(gys) + 1

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

        fig.update_layout(
            width=1400,
            height=700,
            margin=dict(l=20, r=20, t=20, b=20),
            shapes=[
                dict(
                    type="rect",
                    xref="x",
                    yref="y",
                    x0=gxs[0],
                    y0=10,
                    x1=gxs[-1],
                    y1=y_lim,
                    fillcolor="red",
                    opacity=0.15,
                    line_width=0,
                    layer="below",
                ),
                dict(
                    type="rect",
                    xref="x",
                    yref="y",
                    x0=gxs[0],
                    y0=2,
                    x1=gxs[-1],
                    y1=4,
                    fillcolor="red",
                    opacity=0.15,
                    line_width=0,
                    layer="below",
                ),
            ],
            xaxis=dict(range=[gxs[0], gxs[-1]]),
            yaxis=dict(range=[2, y_lim]),
        )
        fig.add_hline(y=4, line_dash="dash", line_color="red")
        fig.add_hline(y=10, line_dash="dash", line_color="red")

        return fig.to_image(format="png")
