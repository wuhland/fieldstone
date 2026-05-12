FROM golang:1.22-alpine AS builder
WORKDIR /app
# Extensions are standalone modules — they do not share fieldstone's go.mod.
# Initialize a new module before building.
COPY go.mod go.sum* ./
RUN go mod download 2>/dev/null || true
COPY . .
RUN CGO_ENABLED=0 go build -o extension .

FROM alpine:3.19
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /app/extension .
EXPOSE 9000
CMD ["./extension"]
