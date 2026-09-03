FROM golang:1.27-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/zaval .
# A fresh volume inherits the ownership of the directory it covers, and the
# process runs as nonroot: without this the first start cannot create the file.
RUN mkdir -p /out/data

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/zaval /zaval
COPY --from=build --chown=65532:65532 /out/data /data
ENV ADDR=:8080 DB_PATH=/data/dayboard.db BACKUP_DIR=/data/backups TZ=Europe/Kyiv
VOLUME /data
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/zaval"]
