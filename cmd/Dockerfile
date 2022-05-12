FROM golang:1.17-alpine AS build_base
WORKDIR /go/iv2
COPY go.mod go.sum ./
RUN go mod download

FROM build_base as service_builder
COPY . /go/iv2
WORKDIR /go/iv2
RUN go build cmd/main.go

FROM golang:1.17-alpine AS service
WORKDIR /go/iv2
COPY --from=service_builder /go/iv2 .

CMD ./main -f config.yaml