# Card Set Studio - Multi-stage Dockerfile
# Builds both Go backend and Next.js frontend

# ============================================
# Stage 1: Build Go backend
# ============================================
FROM golang:1.22-alpine AS backend-builder

WORKDIR /build

# Install build dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY backend/go.mod backend/go.sum ./
RUN go mod download

# Copy source
COPY backend/ .

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -o /studio ./cmd/studio

# ============================================
# Stage 2: Build Next.js frontend
# ============================================
FROM node:20-alpine AS frontend-builder

WORKDIR /build

# Copy package files
COPY frontend/package*.json ./

# Install dependencies
RUN npm ci

# Copy source
COPY frontend/ .

# Set API URL for build
ENV NEXT_PUBLIC_API_URL=/api

# Build
RUN npm run build

# ============================================
# Stage 3: Production image
# ============================================
FROM alpine:3.19

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache \
    nodejs \
    npm \
    ca-certificates \
    tzdata

# Copy Go binary
COPY --from=backend-builder /studio /app/studio

# Copy Next.js standalone build
COPY --from=frontend-builder /build/.next/standalone /app/frontend
COPY --from=frontend-builder /build/.next/static /app/frontend/.next/static
COPY --from=frontend-builder /build/public /app/frontend/public

# Create output directories
RUN mkdir -p /app/output /app/dixit-cards

# Environment defaults
ENV PORT=3001
ENV OLLAMA_URL=http://ollama:11434
ENV SD_URL=http://stable-diffusion:7860
ENV OUTPUT_PATH=/app/output
ENV EXPORT_PATH=/app/dixit-cards
ENV NODE_ENV=production

# Expose ports
EXPOSE 3001 3002

# Create startup script
RUN echo '#!/bin/sh' > /app/start.sh && \
    echo 'echo "Starting Card Set Studio..."' >> /app/start.sh && \
    echo '# Start backend' >> /app/start.sh && \
    echo '/app/studio &' >> /app/start.sh && \
    echo '# Start frontend' >> /app/start.sh && \
    echo 'cd /app/frontend && node server.js &' >> /app/start.sh && \
    echo '# Wait for any process to exit' >> /app/start.sh && \
    echo 'wait -n' >> /app/start.sh && \
    echo '# Exit with status of process that exited first' >> /app/start.sh && \
    echo 'exit $?' >> /app/start.sh && \
    chmod +x /app/start.sh

CMD ["/app/start.sh"]
