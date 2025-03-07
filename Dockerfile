FROM golang:1.24.0-alpine AS build

WORKDIR /app
COPY . .
RUN go mod download
RUN CGO_ENABLED=0 go build -o ./bin/regnotify .

FROM scratch

WORKDIR /etc/regnotify
COPY --from=build /app/bin/regnotify /regnotify

CMD ["/regnotify"]
