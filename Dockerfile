FROM golang:1.25.3-alpine AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /minigate ./cmd/minigate

FROM alpine:3.22
RUN apk add --no-cache ca-certificates && adduser -D -u 10001 minigate
USER minigate
COPY --from=build /minigate /usr/local/bin/minigate
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/minigate"]
