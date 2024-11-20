FROM golang:alpine

RUN apk add --no-cache exiftool

WORKDIR /app

COPY . .

RUN go build ./cmd/clcli/clcli.go

ENTRYPOINT ["./clcli"]
