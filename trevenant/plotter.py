import matplotlib.pyplot as plt
import matplotlib.dates as md
import seaborn as sns

from pytz import timezone
from datetime import datetime
from ghastly.proto.ghastly_pb2 import FileResponse
from ghastly.proto.ghastly_pb2_grpc import PlotterServicer as ps
from io import BytesIO

from store import Store

sns.set_theme(style="darkgrid")

# TODO:
# - add logging
# - restructure code
# - add unit tests


class PlotterServicer(ps):
    def __init__(self) -> None:
        self.store = Store()

    def PlotDaily(self, request, context):
        eastern = timezone("US/Eastern")
        xs = [
            eastern.localize(
                datetime.fromtimestamp(tp.time.seconds + tp.time.nanos / 1e9)
            )
            for tp in request.tps
        ]
        ys = [tp.value for tp in request.tps]
        fname = "daily-" + xs[-1].strftime("%m%d%Y-%H%M%S-%z") + ".png"
        print(fname)
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
    return buf.read()
