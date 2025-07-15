FROM golang:1.21 as builder
WORKDIR /app
COPY . .
RUN go build -o /app/controller main.go

FROM gcr.io/distroless/static:nonroot
COPY --from=builder /app/controller /controller
ENTRYPOINT ["/controller"]
