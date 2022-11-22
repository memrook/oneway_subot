## Build
FROM golang-1.18-alpine AS build

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY . .

RUN go build -o /supportBot

## Deploy
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /

COPY --from=build /supportBot /supportBot
COPY --from=build /settings.json /settings.json

##EXPOSE 8080

##USER nonroot:nonroot

ENTRYPOINT ["/supportBot"]