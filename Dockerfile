FROM golang:1.10

RUN mkdir -p /app

WORKDIR /app

RUN go get -u github.com/cloudflare/cloudflare-go
RUN go get -u github.com/BurntSushi/toml
RUN go get -u github.com/tcnksm/go-httpstat
RUN go get -u github.com/influxdata/influxdb1-client/v2

ADD . /app

RUN go build ./interlockd.go

CMD ["./interlockd"]