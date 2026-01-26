FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS builder
ARG TARGETOS TARGETARCH
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o trap-smtp .

FROM scratch
LABEL org.opencontainers.image.source="https://github.com/probitas-test/trap-servers"
LABEL org.opencontainers.image.description="SMTP trap server that captures and stores emails in memory"
LABEL org.opencontainers.image.licenses="MIT"
COPY --from=builder /app/trap-smtp /trap-smtp
EXPOSE 8080 2525
ENTRYPOINT ["/trap-smtp"]
