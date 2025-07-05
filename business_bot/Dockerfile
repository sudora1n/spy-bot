FROM bufbuild/buf:1.53.0 AS buf-gen
WORKDIR /workspace

COPY --from=root proto ./proto
COPY --from=root buf.gen.yaml buf.yaml ./

RUN buf generate --template buf.gen.yaml


FROM golang:1.24.3-alpine3.21 AS builder

WORKDIR /business_bot

COPY go.mod go.sum ./
RUN go mod download

COPY --from=buf-gen /workspace/business_bot/pb ./pb

COPY . .

RUN CGO_ENABLED=0 \
    go build -ldflags='-s -w -extldflags "-static"' \
    -o /business_bot/appbin cmd/main.go


FROM alpine:3.21

RUN apk add --no-cache ca-certificates
RUN addgroup -g 101 appgroup && adduser -D -u 101 -G appgroup appuser
USER appuser
WORKDIR /home/appuser/business_bot

COPY --from=builder /business_bot/appbin ./

EXPOSE 8080
CMD ["./appbin"]
