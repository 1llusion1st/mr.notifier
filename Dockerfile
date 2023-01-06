FROM golang:1.19

WORKDIR /src
COPY go.sum go.mod main.go /src/
RUN go mod download

COPY notifier /src/notifier/

RUN ls /src
RUN go mod tidy
RUN go build -o /mr.notifier .

ENTRYPOINT ["/mr.notifier"]