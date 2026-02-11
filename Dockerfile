FROM golang:1.26-alpine AS build-stage

RUN apk add --no-cache git

WORKDIR /usr/src/app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o /app/postalion -v -ldflags="-X 'github.com/denyskon/postalion/version.version=$(git describe --tags)'"

FROM alpine:latest AS build-release-stage

COPY --from=build-stage /app/postalion /app/postalion
COPY templates /app/templates

VOLUME /app/templates

WORKDIR /app

EXPOSE 8080

ENTRYPOINT ["/app/postalion"]
