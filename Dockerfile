FROM golang:1.21-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /detector-service .

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
COPY --from=build /detector-service /usr/local/bin/detector-service
WORKDIR /app
VOLUME /app/data
EXPOSE 8080
ENTRYPOINT ["detector-service"]
CMD ["-config", "/app/config.yaml"]
