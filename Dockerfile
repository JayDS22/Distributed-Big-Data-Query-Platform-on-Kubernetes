FROM golang:1.21-alpine AS builder

WORKDIR /build

# Copy source code
COPY . .

# Build CLI
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /build/dbqp .

# Build operator
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /build/operator ./operator

# Final stage
FROM alpine:3.18

# Install ca-certificates for HTTPS
RUN apk add --no-cache ca-certificates curl kubectl

# Create non-root user
RUN addgroup -g 1000 -S dbqp && \
    adduser -S -G dbqp -u 1000 dbqp

WORKDIR /home/dbqp

# Copy binaries
COPY --from=builder /build/dbqp /usr/local/bin/
COPY --from=builder /build/operator /usr/local/bin/

# Copy configuration
COPY --chown=dbqp:dbqp k8s/ /etc/dbqp/

USER dbqp

ENTRYPOINT ["dbqp"]
CMD ["--help"]
