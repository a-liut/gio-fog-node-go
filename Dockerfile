FROM balenalib/raspberry-pi-debian:latest

LABEL Name=gio-fog-node-go Version=0.1.0

WORKDIR /app

RUN apt-get update && apt-get install -y build-essential \
							wget && \
							apt-get clean && \
							rm -rf /var/lib/apt/lists/*

RUN wget -O go.tar.gz https://github.com/hypriot/golang-armbuilds/releases/download/v1.7.3/go1.7.3.linux-armv7.tar.gz && \
		tar -xvf go.tar.gz -C /usr/local

ENV PATH="/usr/local/go/bin:${PATH}"
ENV GOPATH /app

ADD . /app

RUN go get github.com/paypal/gatt

RUN go install app

ENTRYPOINT ["bin/app"]
CMD []
