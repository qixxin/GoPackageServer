FROM golang:latest

RUN mkdir /app

ADD . /go/src/app

WORKDIR /go/src/app

RUN go mod init 

RUN go build -o main .

EXPOSE 8080

CMD ["./main"]