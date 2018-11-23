FROM golang:1.8

WORKDIR /go/src/screach
COPY ./src/ .
RUN ls /go/src/screach

RUN go get -d -v ./...
RUN go install -v ./...

CMD ["screach"]