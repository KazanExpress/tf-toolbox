# syntax=docker/dockerfile:1

##
## Build
##
FROM golang:1.20 AS build

WORKDIR /app

COPY go.mod ./
# COPY go.sum ./
RUN go mod download

COPY ./ ./

ENV GOOS linux
ENV GOARCH amd64

RUN go build github.com/KazanExpress/tf-toolbox/cmd/cleanplan
RUN go build github.com/KazanExpress/tf-toolbox/cmd/findroot

RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=1 go build -ldflags "-linkmode 'external' -extldflags '-static'" github.com/KazanExpress/tf-toolbox/cmd/unlock

##
## Deploy
##
FROM alpine:3.13

WORKDIR /app

COPY --from=build /app/findroot /app/findroot
COPY --from=build /app/cleanplan /app/cleanplan
COPY --from=build /app/unlock /app/unlock


ENV PATH="/app:$PATH"

ENTRYPOINT ["/app/unlock"]
