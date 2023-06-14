FROM golang:1.20.4-alpine3.16

RUN mkdir /app




WORKDIR /app

COPY . /app

RUN go mod tidy

RUN go build -o main .

EXPOSE 3000

CMD ["./main"]