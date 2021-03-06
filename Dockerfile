FROM golang:alpine as build

RUN apk update
RUN apk add git
WORKDIR /app
ADD . /app
RUN go build -ldflags="-s -w" main.go
EXPOSE 16745
FROM alpine:latest
COPY --from=build /app /app
WORKDIR /app
CMD /app/main
