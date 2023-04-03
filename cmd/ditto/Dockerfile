FROM ubuntu:jammy

RUN apt-get -y update
RUN apt-get -y install curl
RUN apt-get -y install wget

COPY .env .
COPY Taskfile.yaml .
COPY ./scripts/install.sh .
COPY mongo-backup.yaml .

RUN ./install.sh

RUN wget https://github.com/gjcarneiro/yacron/releases/download/0.19.0/yacron-0.19.0-x86_64-unknown-linux-gnu -O yacron
RUN chmod +x yacron

# docker run -dit --net=host --mount type=bind,source="$(pwd)"/backups,target=/backups ditto
ENTRYPOINT [ "./yacron", "-c", "mongo-backup.yaml" ]