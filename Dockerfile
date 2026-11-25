FROM golang:1.26-alpine AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/app ./cmd/app/main

FROM alpine:3.22

RUN apk add --no-cache ca-certificates tzdata \
	&& adduser -D -h /app app \
	&& mkdir -p /data \
	&& chown app:app /data

WORKDIR /app
COPY --from=build /out/app /app/app

USER app
ENV PORT=8080
ENV DATA_FILE=/data/meetings.json
EXPOSE 8080
VOLUME /data

CMD ["/app/app"]
