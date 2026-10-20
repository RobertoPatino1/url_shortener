FROM golang:1.24-alpine AS build
ENV GOTOOLCHAIN=auto
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/shortener .

FROM alpine:3.20
RUN adduser -D -u 10001 app
USER app
COPY --from=build /out/shortener /shortener
EXPOSE 8080
ENTRYPOINT ["/shortener"]