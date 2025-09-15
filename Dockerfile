FROM golang:alpine as builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod tidy

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -gcflags="all=-N -l" -ldflags="-s -w" -v -o broker *.go

FROM scratch

COPY --from=builder /app/broker /broker
ENTRYPOINT ["/broker"]
