FROM golang:1.18-alpine AS build_base
WORKDIR /go/iv2
COPY go.mod go.sum ./
RUN go mod download

ARG config_file=config.yaml

FROM build_base as service_builder
COPY gourgeist /go/iv2/gourgeist/
COPY cmd/gourgeist/main.go /go/iv2/cmd/main.go
COPY ${config_file} /go/iv2/${config_file}
WORKDIR /go/iv2
RUN go build cmd/main.go

FROM golang:1.17-alpine AS service
WORKDIR /go/iv2
COPY --from=service_builder /go/iv2 .

CMD ./main -f config.yaml