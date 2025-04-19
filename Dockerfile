# ---------- 1) Build stage ---------------------------------------------------
FROM golang:1.23-alpine AS build

RUN apk add --no-cache git ca-certificates && update-ca-certificates
WORKDIR /src

# 1) Copy go.mod and go.sum
COPY go.mod go.sum ./
RUN go mod download

# 2) Copy everything else in backend/
COPY . ./

# 3) Build the Go binary
RUN CGO_ENABLED=0 \
    go build -ldflags="-s -w" -o /src/backend ./cmd

# ---------- 2) Runtime stage -------------------------------------------------
FROM gcr.io/distroless/base-debian12
WORKDIR /app

COPY --from=build /src/backend /app/backend
COPY --from=build /src/assets  /app/assets
COPY --from=build /src/fonts   /app/fonts
COPY --from=build /src/json    /app/json

EXPOSE 8080
ENTRYPOINT ["/app/backend"]

