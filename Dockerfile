FROM golang:1.24.1-alpine3.21 AS builder

COPY ${PWD} /app
WORKDIR /app

RUN CGO_ENABLED=0 go build -ldflags '-s -w -extldflags "-static"' -o /app/appbin cmd/main.go

FROM alpine:3.21
RUN apk --update add ca-certificates && \
    rm -rf /var/cache/apk/*

RUN addgroup -g 101 appgroup && \
    adduser -D -u 101 -G appgroup appuser
USER appuser

COPY --from=builder /app /home/appuser/app

WORKDIR /home/appuser/app

EXPOSE 8079
EXPOSE 8080

CMD ["./appbin"]
