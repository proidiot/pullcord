FROM golang:1.6

COPY . /go/src/github.com/stuphlabs/pullcord

WORKDIR /go/src/github.com/stuphlabs/pullcord

RUN go get -d -v ./...
RUN go install -v ./main

CMD ["main"]

