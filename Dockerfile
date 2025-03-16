# Use the official Go image as the base image
FROM golang:1.24 as builder

# Set the working directory to /app
WORKDIR /app

# Copy the Go module files and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the application code
COPY . .

# Build the Go application
RUN go build -o ekz-tesla .

# Use a minimal base image for the final image
FROM scratch

# Set the working directory to /app
WORKDIR /app

# Copy the built Go application from the builder stage
COPY --from=builder /app/ekz-tesla .

# Set the entry point to the executable
ENTRYPOINT ["./ekz-tesla"]
