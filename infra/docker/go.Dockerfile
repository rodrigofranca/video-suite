# API and worker images. Build context: the repository root (Go workspace).
#   docker build -f infra/docker/go.Dockerfile --target api .
#   docker build -f infra/docker/go.Dockerfile --target worker .

FROM golang:1.25.14-alpine3.23 AS build
WORKDIR /src
# Module files first, so dependencies are cached between code changes.
COPY go.work go.work.sum ./
COPY packages/job-contracts/go.mod packages/job-contracts/
COPY packages/media-core/go.mod packages/media-core/go.sum packages/media-core/
COPY apps/api/go.mod apps/api/go.sum apps/api/
COPY apps/worker/go.mod apps/worker/go.sum apps/worker/
RUN go mod download
COPY packages packages
COPY apps/api apps/api
COPY apps/worker apps/worker
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/api ./apps/api/cmd/api \
 && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/worker ./apps/worker/cmd/worker

FROM alpine:3.23.8 AS api
# The base image already has the CA bundle; the API needs no extra packages.
RUN adduser -D -H app
COPY --from=build /out/api /usr/local/bin/api
USER app
EXPOSE 8080
ENTRYPOINT ["api"]

FROM alpine:3.23.8 AS worker
# FFmpeg (with libx264 and ffprobe) does the media processing.
RUN apk add --no-cache ca-certificates ffmpeg && adduser -D -H app
COPY --from=build /out/worker /usr/local/bin/worker
USER app
ENTRYPOINT ["worker"]
