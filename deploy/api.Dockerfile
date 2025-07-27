FROM golang:1.24.4-alpine3.21 AS builder

WORKDIR /workspace

RUN --mount=type=cache,target=/go/pkg/mod/ \
    --mount=type=bind,source=business_bot/go.sum,target=business_bot/go.sum \
    --mount=type=bind,source=business_bot/go.mod,target=business_bot/go.mod \
    --mount=type=bind,source=creator_bot/go.mod,target=creator_bot/go.mod \
    --mount=type=bind,source=creator_bot/go.sum,target=creator_bot/go.sum \
    --mount=type=bind,source=api/go.mod,target=api/go.mod \
    --mount=type=bind,source=api/go.sum,target=api/go.sum \
    --mount=type=bind,source=proto/,target=proto/ \
    --mount=type=bind,source=common/,target=common/ \
    cd api && go mod download -x

RUN --mount=type=cache,target=/go/pkg/mod/ \
    --mount=type=bind,target=. \
    CGO_ENABLED=0 go build -ldflags='-s -w -extldflags "-static"' -o /bin/api ./api/cmd/main.go


FROM alpine:3.21

RUN apk add --no-cache ca-certificates
RUN addgroup -g 101 appgroup && adduser -D -u 101 -G appgroup appuser
USER appuser
WORKDIR /home/appuser/api

COPY --from=builder /bin/api ./

EXPOSE 8080
CMD ["./api"]
