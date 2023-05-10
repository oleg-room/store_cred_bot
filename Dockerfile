FROM golang:latest

WORKDIR /go/src
COPY . .
RUN go build
ENTRYPOINT ["go", "run", "."]



