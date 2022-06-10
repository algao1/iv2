import gc
import matplotlib.pyplot as plt
import matplotlib.dates as md
import seaborn as sns

from datetime import datetime
from ghastly.proto.ghastly_pb2 import FileResponse
from ghastly.proto.ghastly_pb2_grpc import PlotterServicer as ps
from io import BytesIO
from loguru import logger
from pytz import timezone

from store import Store

sns.set_theme(style="darkgrid")

# TODO:
# - add unit tests
# - add type hints
# - restructure code


class PlotterServicer(ps):
    def __init__(self, uri="mongodb://localhost:27017/") -> None:
        self.store = Store(uri)

    def PlotDaily(self, request, context):
        logger.debug("got request to generate daily plot")

        eastern = timezone("US/Eastern")
        xs = [
            eastern.localize(
                datetime.fromtimestamp(glucose.time.seconds + glucose.time.nanos / 1e9)
            )
            for glucose in request.glucose
        ]
        ys = [glucose.value for glucose in request.glucose]
        fname = "daily-" + xs[-1].strftime("%m%d%Y-%H%M%S-%z") + ".png"
        iid = self.store.store_image(plot(xs, ys), fname)

        return FileResponse(id=f"{iid}", name=fname)


def plot(xs, ys):
    tz = timezone("US/Eastern")
    fig, ax = plt.subplots(figsize=(20, 8))
    ax.set(ylim=(3, 20))

    sns.lineplot(x=xs, y=ys)

    major_xlocator = md.HourLocator(interval=1, tz=tz)
    major_xformatter = md.DateFormatter("%I:%M%p", tz=tz)

    ax.xaxis.set_major_locator(major_xlocator)
    fig.axes[0].xaxis.set_major_formatter(major_xformatter)

    buf = BytesIO()
    plt.savefig(buf, bbox_inches="tight")
    buf.seek(0)

    plt.cla()
    plt.clf()
    plt.close("all")
    gc.collect()

    return buf.read()
