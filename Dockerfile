FROM golang:1.27-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /zaval .

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata sqlite
ENV ADDR=:8080 DB_PATH=/data/dayboard.db TZ=Europe/Kyiv
VOLUME /data
COPY --from=build /zaval /usr/local/bin/zaval
COPY deploy/backup.sh /usr/local/bin/backup
EXPOSE 8080
CMD ["zaval"]
