FROM golang:1.26-alpine AS build
ARG VERSION=0.0.0-dev
ARG GIT_COMMIT=unknown
ARG BUILD_DATE=unknown
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN mkdir -p /out && \
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w -X idmagic/backend/shared/version.Version=${VERSION} -X idmagic/backend/shared/version.GitCommit=${GIT_COMMIT} -X idmagic/backend/shared/version.BuildDate=${BUILD_DATE}" -o /out/idmagic ./backend/cmd/idmagic && \
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w -X idmagic/backend/shared/version.Version=${VERSION} -X idmagic/backend/shared/version.GitCommit=${GIT_COMMIT} -X idmagic/backend/shared/version.BuildDate=${BUILD_DATE}" -o /out/idmagic-relay ./backend/cmd/idmagic-relay && \
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w -X idmagic/backend/shared/version.Version=${VERSION} -X idmagic/backend/shared/version.GitCommit=${GIT_COMMIT} -X idmagic/backend/shared/version.BuildDate=${BUILD_DATE}" -o /out/idmagic-worker ./backend/cmd/idmagic-worker

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=build /out/idmagic /app/idmagic
COPY --from=build /out/idmagic-relay /app/idmagic-relay
COPY --from=build /out/idmagic-worker /app/idmagic-worker
COPY spec /app/spec
COPY deploy/schema /app/deploy/schema
EXPOSE 8080
ENTRYPOINT ["/app/idmagic"]
