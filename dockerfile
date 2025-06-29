# BUILDTIME
FROM golang:1.25-rc-bookworm AS builder
ENV GO111MODULE=on CGO_ENABLED=0
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -trimpath -ldflags="-s -w" -o ./server ./main.go


# RUNTIME
FROM gcr.io/distroless/static:nonroot
COPY --from=builder /app/server /server
USER nonroot:nonroot
HEALTHCHECK NONE
ENTRYPOINT ["/server"]
