FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /bin/sre-agent ./cmd/agent

FROM alpine:3.20
RUN adduser -D -u 10001 appuser
COPY --from=build /bin/sre-agent /bin/sre-agent
COPY prompts/ /home/appuser/prompts/
USER appuser
WORKDIR /home/appuser
EXPOSE 8080
HEALTHCHECK --interval=10s --timeout=3s --start-period=5s --retries=3 \
  CMD wget -qO- http://localhost:8080/api/v1/healthz || exit 1
ENTRYPOINT ["/bin/sre-agent"]
