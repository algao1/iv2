import grpc
import yaml

from concurrent import futures
from loguru import logger
from modules.plotter import PlotterServicer

from ghastly.proto.ghastly_pb2_grpc import add_PlotterServicer_to_server
from modules.store import Store


if __name__ == "__main__":
    with open("config.yaml", "r") as file:
        config = yaml.safe_load(file)

    store = Store(config["mongo"]["uri"])
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
    add_PlotterServicer_to_server(PlotterServicer(config, store), server)

    server.add_insecure_port("[::]:50051")
    logger.debug("started server on port 50051")

    server.start()
    server.wait_for_termination()
