# Multi-stage Dockerfile for Dixit game
# Stage 1: Build Go backend
FROM golang:1.21-alpine AS backend-builder

WORKDIR /build

# Copy all backend source code
COPY backend/ .

# Generate go.sum and download dependencies
RUN go mod tidy

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/server

# Stage 2: Build Next.js frontend
FROM node:20-alpine AS frontend-builder

WORKDIR /build

# Copy package files
COPY frontend/package.json frontend/package-lock.json* ./

# Install dependencies
RUN npm install

# Copy source code
COPY frontend/ .

# Build Next.js
RUN npm run build

# Stage 3: Production runtime
FROM alpine:3.19

# Install Node.js and supervisord
RUN apk add --no-cache nodejs npm supervisor

# Create app directory
WORKDIR /app

# Copy Go backend binary
COPY --from=backend-builder /build/server /app/backend/server

# Copy Next.js standalone build
COPY --from=frontend-builder /build/.next/standalone /app/frontend/.next/standalone
COPY --from=frontend-builder /build/.next/static /app/frontend/.next/standalone/.next/static
COPY --from=frontend-builder /build/public /app/frontend/.next/standalone/public

# Copy cards directory
COPY cards/ /app/cards/

# Copy supervisord config
COPY docker/supervisord.conf /etc/supervisor/conf.d/supervisord.conf

# Create log directory
RUN mkdir -p /var/log

# Expose port 8080 (Go backend serves as entry point)
EXPOSE 8080

# Environment variables
ENV PORT=8080
ENV CARDS_PATH=/app/cards
ENV NEXTJS_URL=http://localhost:3000
ENV MIN_PLAYERS=4

# Start supervisord
CMD ["/usr/bin/supervisord", "-c", "/etc/supervisor/conf.d/supervisord.conf"]
