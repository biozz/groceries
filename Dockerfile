FROM golang:1.17-alpine as builder
ENV CGO_ENABLED=0
WORKDIR /app
COPY . .
RUN go build -o bin/groceries .

FROM alpine:latest
COPY --from=builder /app/bin /usr/local/bin
ENTRYPOINT ["groceries"]
