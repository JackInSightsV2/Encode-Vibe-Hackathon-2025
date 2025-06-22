# Multi-stage Dockerfile for QT-1 Middleware
# Stage 1: Build Frontend with Node.js
FROM node:18-alpine AS frontend-builder

# Set working directory for frontend
WORKDIR /app/frontend

# Copy frontend package files
COPY frontend/package*.json ./

# Install dependencies
RUN npm install

# Copy frontend source code
COPY frontend/ ./

# Build the React application
RUN npm run build

# Stage 2: Build and Run Backend with Go
FROM golang:1.23-alpine AS backend-builder

# Install git (needed for some Go modules)
RUN apk add --no-cache git

# Set working directory for backend
WORKDIR /app/backend

# Copy go mod files first for better caching
COPY backend/go.mod backend/go.sum ./

# Download dependencies
RUN go mod download

# Copy backend source code
COPY backend/ ./

# Build the Go application
RUN CGO_ENABLED=0 GOOS=linux go build -o main .

# Stage 3: Runtime
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

# Create app directory
WORKDIR /app

# Create the backend directory structure
RUN mkdir -p /app/backend /app/frontend/dist /app/logs /app/storage /app/backend/database/migrations

# Copy the built Go binary
COPY --from=backend-builder /app/backend/main /app/backend/

# Copy backend configuration and other necessary files
COPY --from=backend-builder /app/backend/config.yaml /app/backend/
COPY --from=backend-builder /app/backend/.env /app/backend/
COPY --from=backend-builder /app/backend/rules/ /app/backend/rules/
COPY --from=backend-builder /app/backend/database/migrations/ /app/backend/database/migrations/

# Copy the built frontend dist
COPY --from=frontend-builder /app/frontend/dist/ /app/frontend/dist/

# Set working directory to backend (so ../frontend/dist resolves correctly)
WORKDIR /app/backend

# Expose the port (default is typically 8080, adjust if different)
EXPOSE 8080

# Run the application as root (required for Azure Container Registry)
CMD ["./main"] 