# multistage docker build. This redices the size of final docker image
# stage 1 to build the app
FROM golang:alpine as builder

ADD . /build/

WORKDIR /build

RUN go build -o main .

# stage 2 deploys the app built in stage 1
FROM alpine

RUN adduser -S -D -H -h /app appuser

COPY --from=builder --chown=user:appuser /build/main /app/

USER appuser

WORKDIR /app

EXPOSE 3000

CMD ["./main"]
