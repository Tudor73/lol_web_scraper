FROM golang:1.22

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

ADD . /app

RUN CGO_ENABLED=0 GOOS=linux go build -o /exec


EXPOSE 8080
CMD ["/exec"]
