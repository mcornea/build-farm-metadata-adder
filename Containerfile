# ---- Build Stage ----
# Use the official Go image as a builder
FROM --platform=$BUILDPLATFORM golang:1.24-alpine AS builder

ARG TARGETARCH
ARG TARGETOS

# Set the working directory
WORKDIR /app

# Copy go.mod and go.sum to leverage Docker cache
COPY go.mod go.sum ./
# Download dependencies
RUN go mod download

# Copy the source code
COPY main.go .

# Build the Go app, creating a static binary
# CGO_ENABLED=0 is critical for building a static binary that can run in a scratch image
# -ldflags="-w -s" strips debug symbols to reduce the binary size
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -a -ldflags="-w -s" -o metadata-adder .

# ---- Final Stage ----
# Use a minimal, empty base image
FROM scratch

# Copy only the compiled binary from the builder stage
COPY --from=builder /app/metadata-adder /metadata-adder

# Set the entrypoint for the container
ENTRYPOINT ["/metadata-adder"]
