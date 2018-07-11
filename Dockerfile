FROM golang:1.10

RUN go get github.com/Masterminds/glide

ENV BROKER demo.nats.io:4222
ENV NAME demo

WORKDIR /go/src/github.com/choria-io/go-backplane
COPY . .

RUN glide install
RUN go build -o /backplane
RUN cd example;go build -o /backplane-example
RUN cp example/myapp.yaml.example /myapp.yaml

ENTRYPOINT ["./container-start.sh"]