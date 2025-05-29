FROM golang:1.24.1-alpine3.21 AS builder

COPY --from=root Makefile /
COPY --from=root proto /proto
WORKDIR /

RUN apk add --no-cache make protoc

COPY . /creator_bot/

RUN make install-tools && make generate-creator

WORKDIR /creator_bot
RUN CGO_ENABLED=0 go build -ldflags '-s -w -extldflags "-static"' -o /creator_bot/appbin cmd/main.go

WORKDIR /
RUN make generate-creator

FROM alpine:3.21
RUN apk --update add ca-certificates && \
    rm -rf /var/cache/apk/*

RUN addgroup -g 101 appgroup && \
    adduser -D -u 101 -G appgroup appuser
USER appuser

COPY --from=builder /creator_bot /home/appuser/creator_bot

WORKDIR /home/appuser/creator_bot

EXPOSE 8080

CMD ["./appbin"]
