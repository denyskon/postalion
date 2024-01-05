FROM golang:1.21-alpine AS build-stage

WORKDIR /usr/src/app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o /app/postalion

COPY templates /app/templates

VOLUME /app/templates

WORKDIR /app

EXPOSE 8080

ENTRYPOINT ["/app/postalion"]