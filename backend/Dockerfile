# ---- build stage ----
FROM golang:1.24-alpine AS build
WORKDIR /app
RUN apk add --no-cache git
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /server ./cmd/server

# ---- runtime stage ----
FROM alpine:3.20
RUN apk add --no-cache ca-certificates && adduser -D -u 10001 appuser
COPY --from=build /server /server
USER appuser
EXPOSE 8080
ENTRYPOINT ["/server"]
