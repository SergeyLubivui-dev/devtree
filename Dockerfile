# devtree in a container: generate the diagrams without installing Go.
#
#   docker run --rm -v "$PWD:/work" ghcr.io/sergeylubivui-dev/devtree render
#
# The build cross-compiles rather than emulating: the compiler stage always
# runs on the machine doing the building ($BUILDPLATFORM) and is told what to
# produce, which is the difference between a two-minute build and a twenty-
# minute one when a multi-arch image is being assembled.

FROM --platform=$BUILDPLATFORM golang:1.24-alpine AS build

WORKDIR /src

# The module has no dependencies, so there is nothing to download and no
# separate layer worth caching for it. Copy the source and build.
COPY . .

ARG TARGETOS
ARG TARGETARCH
ARG VERSION=dev

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
      -trimpath \
      -ldflags "-s -w -X github.com/SergeyLubivui-dev/devtree/internal/cli.Version=${VERSION}" \
      -o /out/devtree .

# Alpine rather than scratch, for one reason: `devtree sync` reads what git
# already knows, and a scratch image has no git to read it with. Everything
# else in devtree works on the file alone, so the extra few megabytes buy the
# one command that would otherwise be missing.
FROM alpine:3.22

RUN apk add --no-cache git

COPY --from=build /out/devtree /usr/local/bin/devtree

# /work is where the repository gets mounted. Nothing is baked into the image.
WORKDIR /work

# git refuses to work in a directory owned by someone else, which is exactly
# what a bind-mounted repository looks like from inside a container.
RUN git config --global --add safe.directory '*'

ENTRYPOINT ["devtree"]
CMD ["help"]

LABEL org.opencontainers.image.title="devtree" \
      org.opencontainers.image.description="Tree-shaped development planning that lives inside your repository" \
      org.opencontainers.image.source="https://github.com/SergeyLubivui-dev/devtree" \
      org.opencontainers.image.licenses="MIT"
