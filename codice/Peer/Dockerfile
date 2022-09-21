# syntax=docker/dockerfile:1
FROM golang:1.17-alpine
ADD . /app
WORKDIR /app
COPY . .
RUN go mod download
RUN go build
ENTRYPOINT ["./main"]
