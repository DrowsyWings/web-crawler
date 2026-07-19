# syntax=docker/dockerfile:1

FROM golang:1.26 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/web-crawler .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/web-crawler /usr/local/bin/web-crawler
ENTRYPOINT ["/usr/local/bin/web-crawler"]
