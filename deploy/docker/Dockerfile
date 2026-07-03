FROM golang:1.26-alpine AS build
ARG VERSION=0.0.0-dev
ARG GIT_COMMIT=unknown
ARG BUILD_DATE=unknown
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN mkdir -p /out && \
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w -X idmagic/internal/shared/version.Version=${VERSION} -X idmagic/internal/shared/version.GitCommit=${GIT_COMMIT} -X idmagic/internal/shared/version.BuildDate=${BUILD_DATE}" -o /out/idmagic ./cmd/idmagic && \
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w -X idmagic/internal/shared/version.Version=${VERSION} -X idmagic/internal/shared/version.GitCommit=${GIT_COMMIT} -X idmagic/internal/shared/version.BuildDate=${BUILD_DATE}" -o /out/idmagic-relay ./cmd/idmagic-relay

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=build /out/idmagic /app/idmagic
COPY --from=build /out/idmagic-relay /app/idmagic-relay
COPY spec /app/spec
COPY deploy/schema /app/deploy/schema
EXPOSE 8080
ENTRYPOINT ["/app/idmagic"]
