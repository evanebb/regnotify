FROM golang:1.24.4-alpine AS build

WORKDIR /app
COPY . .
RUN go mod download
RUN CGO_ENABLED=0 go build -o ./bin/regnotify ./cmd/regnotify

FROM scratch

WORKDIR /var/lib/regnotify
WORKDIR /etc/regnotify
COPY configuration/config-docker.yml ./config.yml
COPY --from=build /app/bin/regnotify /regnotify

CMD ["/regnotify", "serve", "/etc/regnotify/config.yml"]
