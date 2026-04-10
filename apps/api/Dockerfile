FROM golang:1.22-alpine AS build
WORKDIR /src
# Copy go module files first for layer caching
COPY go.mod go.sum ./
RUN go mod download
# Copy source
COPY . .
RUN CGO_ENABLED=0 go build -o /out/api ./apps/api

FROM gcr.io/distroless/static:nonroot
COPY --from=build /out/api /api
COPY --from=build /src/migrations /migrations
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/api"]
