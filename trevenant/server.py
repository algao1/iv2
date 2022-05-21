import grpc

from concurrent import futures
from plotter import PlotterServicer

from ghastly.proto.ghastly_pb2_grpc import add_PlotterServicer_to_server


if __name__ == "__main__":
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
    add_PlotterServicer_to_server(PlotterServicer(), server)
    server.add_insecure_port("[::]:50051")
    server.start()
    server.wait_for_termination()
