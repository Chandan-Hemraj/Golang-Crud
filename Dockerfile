FROM golang:1.17-alpine

WORKDIR /app

RUN go build -o /APP_NAME/$CMD_PATH

