FROM golang:alpine AS builder

WORKDIR /fognode

# Install git for fetching dependencies
RUN apk update && apk add --no-cache git

COPY go.mod .

RUN go mod download

COPY . .

# Build the binary.
RUN go build -o /go/bin/fognode cmd/fog-node/main.go

## Build lighter image
FROM alpine:latest
LABEL Name=gio-fog-node-go Version=1.0.0

# Copy our static executable.
COPY --from=builder /go/bin/fognode /fognode

ARG GIO_FOG_NODE_SERVER_PORT

EXPOSE 8080
EXPOSE $GIO_FOG_NODE_SERVER_PORT

# Run the binary.
ENTRYPOINT /fognode