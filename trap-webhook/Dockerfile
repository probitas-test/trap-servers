FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS builder
ARG TARGETOS TARGETARCH
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o trap-webhook .

FROM scratch
LABEL org.opencontainers.image.source="https://github.com/probitas-test/trap-servers"
LABEL org.opencontainers.image.description="Webhook trap server that captures and stores HTTP requests in memory"
LABEL org.opencontainers.image.licenses="MIT"
COPY --from=builder /app/trap-webhook /trap-webhook
EXPOSE 8080
ENTRYPOINT ["/trap-webhook"]
