FROM golang:1.20-alpine as builder

WORKDIR /app
COPY . ./

RUN go build -o ./z42-resolver ./cmd/resolver/
RUN go build -o ./z42-api ./cmd/api/
RUN go build -o ./z42-updater ./cmd/zone_updater/

FROM alpine as resolver
COPY --from=builder /app/z42-resolver /bin/z42-resolver
ENTRYPOINT ["/bin/z42-resolver", "-c", "/etc/z42/resolver-config.json"]

FROM alpine as api
COPY --from=builder /app/z42-api /bin/z42-api
ENTRYPOINT ["/bin/z42-api", "-c", "/etc/z42/api-config.json"]

FROM alpine as updater
COPY --from=builder /app/z42-updater /bin/z42-updater
ENTRYPOINT ["/bin/z42-updater", "-c", "/etc/z42/updater-config.json"]

