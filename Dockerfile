# syntax=docker/dockerfile:1

FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /casadellibro-mcp ./cmd/app

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /casadellibro-mcp /casadellibro-mcp
# PaaS inject $PORT; the binary binds to it when --addr is not passed.
ENV PORT=8080
EXPOSE 8080
ENTRYPOINT ["/casadellibro-mcp"]
CMD ["serve", "--transport", "http"]
