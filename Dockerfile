FROM golang:1.20-alpine as builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod tidy

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o broker main.go

FROM scratch

COPY --from=builder /app/broker /broker
ENTRYPOINT ["/broker"]
