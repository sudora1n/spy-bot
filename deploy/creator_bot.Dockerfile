FROM golang:1.24.4-alpine3.21 AS builder

WORKDIR /workspace

RUN --mount=type=cache,target=/go/pkg/mod/ \
    --mount=type=bind,source=business_bot/go.sum,target=business_bot/go.sum \
    --mount=type=bind,source=business_bot/go.mod,target=business_bot/go.mod \
    --mount=type=bind,source=creator_bot/go.mod,target=creator_bot/go.mod \
    --mount=type=bind,source=creator_bot/go.sum,target=creator_bot/go.sum \
    --mount=type=bind,source=proto/,target=proto/ \
    --mount=type=bind,source=common/,target=common/ \
    cd creator_bot && go mod download -x

RUN --mount=type=cache,target=/go/pkg/mod/ \
    --mount=type=bind,target=. \
    CGO_ENABLED=0 go build -ldflags='-s -w -extldflags "-static"' -o /bin/creator_bot ./creator_bot/cmd/main.go


FROM alpine:3.21

RUN apk add --no-cache ca-certificates
RUN addgroup -g 101 appgroup && adduser -D -u 101 -G appgroup appuser
USER appuser
WORKDIR /home/appuser/creator_bot

COPY --from=builder /bin/creator_bot ./

EXPOSE 8080
CMD ["./creator_bot"]
