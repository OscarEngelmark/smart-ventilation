# Built from the repository root (see docker-compose.yml), since the service
# needs go.mod and internal/ as well as its own folder.

FROM golang:1.25 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /occupancy-sensor ./cmd/occupancy-sensor

FROM gcr.io/distroless/static-debian12
COPY --from=build /occupancy-sensor /occupancy-sensor
ENTRYPOINT ["/occupancy-sensor"]
