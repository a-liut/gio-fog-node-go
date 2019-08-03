#FROM golang:latest
#
#ADD . /usr/local/go/src/github.com/a-liut/gio-fog-node-go
#
#RUN go get github.com/paypal/gatt
#
#RUN go install gio-fog
#
#ENTRYPOINT ["bin/app"]
#CMD []

FROM golang:alpine AS builder

ARG SRC_ROOT

# Install git for fetching dependencies
RUN apk update && apk add --no-cache git

WORKDIR /go/src/github.com/a-liut/${SRC_ROOT}
ADD . /go/src/github.com/a-liut/${SRC_ROOT}

# Fetch dependencies.
RUN go get -d -v ./...

# Build the binary.
RUN go install ./...

## Build lighter image
FROM alpine:latest

ARG SRC_ROOT
ENV SRC_ROOT=${SRC_ROOT}

# Copy our static executable.
COPY --from=builder /go/bin/${SRC_ROOT} /app

EXPOSE 8080

# Run the binary.
ENTRYPOINT /app