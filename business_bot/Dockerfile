FROM golang:1.24.1-alpine3.21 AS builder

COPY --from=root Makefile /
COPY --from=root proto /proto
WORKDIR /

RUN apk add --no-cache make protoc

COPY . /business_bot/

RUN make install-tools && make generate-business

WORKDIR /business_bot
RUN CGO_ENABLED=0 go build -ldflags '-s -w -extldflags "-static"' -o /business_bot/appbin cmd/main.go

FROM alpine:3.21
RUN apk --update add ca-certificates && \
    rm -rf /var/cache/apk/*

RUN addgroup -g 101 appgroup && \
    adduser -D -u 101 -G appgroup appuser
USER appuser

COPY --from=builder /business_bot /home/appuser/business_bot

WORKDIR /home/appuser/business_bot

EXPOSE 8080

CMD ["./appbin"]
