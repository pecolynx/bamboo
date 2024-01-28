FROM golang:1.21-alpine as builder

RUN apk add --no-cache build-base=0.5-r3

WORKDIR /app
COPY go.mod go.mod ./
RUN go mod download

COPY . .
RUN go build -o calc-app ./

# Application image.
FROM alpine:3.18.6

# RUN apk --no-cache add tzdata=2022f-r0

WORKDIR /app

COPY --from=builder /app/calc-app .

RUN addgroup -S appgroup && \
    adduser -S appuser -G appgroup && \
    chown -R appuser /app

USER appuser

EXPOSE 8081

CMD ["./calc-app"]
