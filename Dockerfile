FROM golang:1.24.2

WORKDIR /app
COPY . .

RUN go mod tidy
RUN go build -o app cmd/node/main.go

CMD ["./app"]