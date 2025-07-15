FROM golang:1.21 as builder
WORKDIR /app
COPY . .
ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64
RUN go build -o /app/controller main.go

FROM gcr.io/distroless/static:nonroot
COPY --from=builder /app/controller /controller
ENTRYPOINT ["/controller"]
