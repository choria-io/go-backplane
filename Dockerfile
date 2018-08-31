FROM golang:1.10

RUN go get github.com/Masterminds/glide

WORKDIR /go/src/github.com/choria-io/go-backplane
COPY . .

RUN glide install
RUN CGO_ENABLED=0 go build -o /backplane -a -ldflags '-extldflags "-static" -w -s' && \
    cd example && \
    CGO_ENABLED=0 go build -o /backplane-example -a -ldflags '-extldflags "-static" -w -s'

FROM alpine:latest

ENV BROKER demo.nats.io:4222
ENV NAME demo

COPY ./container-start.sh .
COPY --from=0 /backplane-example /backplane /


ENTRYPOINT ["./container-start.sh"]