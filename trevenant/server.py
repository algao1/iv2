import grpc
import yaml

from concurrent import futures
from plotter import PlotterServicer

from ghastly.proto.ghastly_pb2_grpc import add_PlotterServicer_to_server


if __name__ == "__main__":
    with open("config.yaml", "r") as file:
        config = yaml.safe_load(file)

    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
    add_PlotterServicer_to_server(PlotterServicer(config["mongoURI"]), server)
    server.add_insecure_port("[::]:50051")
    server.start()
    server.wait_for_termination()
