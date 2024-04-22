FROM golang:1.22.1 AS builder

WORKDIR /go/src/app
COPY . .

RUN go build -o /go/bin/haproxy-ccm

FROM gcr.io/distroless/base
COPY --from=builder /go/bin/haproxy-ccm /
CMD ["/haproxy-ccm"]