FROM python:3.9.13-slim-buster AS build_base

WORKDIR /trevenant

COPY requirements.txt .
RUN pip3 install --upgrade pip
RUN pip3 install -r requirements.txt
COPY docker-config.yaml config.yaml
COPY trevenant .

EXPOSE 50051

CMD python3 server.py