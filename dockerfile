FROM golang:1.19

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY *.go ./

RUN go build -o /docker-gs-ping


EXPOSE 8080
CMD ["/docker-gs-ping"]