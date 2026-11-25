# --- UI -----------------------------------------------------------------------
FROM --platform=$BUILDPLATFORM node:22-alpine AS ui
WORKDIR /ui
COPY ui/package.json ui/package-lock.json ./
RUN npm ci
COPY ui .
RUN npm run build

# --- Go -----------------------------------------------------------------------
FROM --platform=$BUILDPLATFORM golang:1.27-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=ui /ui/dist ./ui/dist
ARG VERSION=dev
ARG TARGETOS TARGETARCH
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags="-s -w -X main.version=${VERSION}" -o /wishlist .

# --- Runtime ------------------------------------------------------------------
FROM alpine:3.21
LABEL org.opencontainers.image.source="https://github.com/floholz/wishlist" \
      org.opencontainers.image.description="Simple wishlists — one link to edit, one to share" \
      org.opencontainers.image.licenses="MIT"
RUN apk add --no-cache ca-certificates tzdata
COPY --from=build /wishlist /wishlist
VOLUME /pb_data
EXPOSE 8080
ENTRYPOINT ["/wishlist", "serve", "--http=0.0.0.0:8080", "--dir=/pb_data"]
