FROM golang:1.17-alpine AS build_base
WORKDIR /go/iv2
COPY go.mod go.sum ./
RUN go mod download

FROM build_base as service_builder
COPY gourgeist /go/iv2/gourgeist/
COPY cmd/gourgeist/main.go /go/iv2/cmd/main.go
COPY docker-config.yaml /go/iv2/config.yaml
WORKDIR /go/iv2
RUN go build cmd/main.go

FROM golang:1.17-alpine AS service
WORKDIR /go/iv2
COPY --from=service_builder /go/iv2 .

CMD ./main -f config.yaml