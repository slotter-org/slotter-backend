# ---------- 1) Build stage ---------------------------------------------------
FROM golang:1.22-alpine AS build

RUN apk add --no-cache git ca-certificates && update-ca-certificates
WORKDIR /src

# cache Go modules first
COPY backend/go.mod backend/go.sum ./
RUN go mod download

# copy source and build
COPY backend/ ./
RUN CGO_ENABLED=0 \
    go build -ldflags="-s -w" -o /src/backend ./cmd        # <- path, not file

# ---------- 2) Runtime stage -------------------------------------------------
FROM gcr.io/distroless/base-debian12
WORKDIR /app

COPY --from=build /src/backend             /app/backend
COPY --from=build /src/assets              /app/assets
COPY --from=build /src/fonts               /app/fonts
COPY --from=build /src/json                /app/json

EXPOSE 8080
ENTRYPOINT ["/app/backend"]

