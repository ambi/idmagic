FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN mkdir -p /out && \
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/idmagic ./cmd/idmagic && \
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/idmagic-relay ./cmd/idmagic-relay

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=build /out/idmagic /app/idmagic
COPY --from=build /out/idmagic-relay /app/idmagic-relay
COPY spec /app/spec
COPY deploy/schema /app/deploy/schema
EXPOSE 8080
ENTRYPOINT ["/app/idmagic"]
