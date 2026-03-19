# Frontend build stage
FROM oven/bun:1-alpine AS builder

WORKDIR /app

# Copy package files and lockfile
COPY package.json bun.lock ./

# Install dependencies
RUN bun install --frozen-lockfile

# Copy source code
COPY . .

# Build the application
RUN bun run build

# Production stage
FROM node:25-alpine

WORKDIR /app

# Copy standalone output from builder
COPY --from=builder /app/.next/standalone ./

# Copy static assets
COPY --from=builder /app/.next/static ./.next/static

# Expose port
EXPOSE 3000

# Set environment to production
ENV NODE_ENV=production
ENV PORT=3000
ENV HOSTNAME="0.0.0.0"

# Run the application
CMD ["node", "server.js"]
