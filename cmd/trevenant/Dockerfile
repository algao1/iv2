FROM python:3.9.13-slim-buster AS build_base

WORKDIR /trevenant

ARG config_file=config.yaml

COPY requirements.txt .
RUN pip3 install --upgrade pip
RUN pip3 install -r requirements.txt
COPY ${config_file} ${config_file}
COPY trevenant /trevenant/

EXPOSE 50051

CMD python3 server.py