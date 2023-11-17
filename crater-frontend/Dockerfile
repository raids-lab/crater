# Phase 1: Build the React Vite app with PNPM
FROM node:20 AS builder

# Set working directory
WORKDIR /app

# Copy only package.json and pnpm-lock.yaml to take advantage of caching
COPY package.json pnpm-lock.yaml ./

# Install dependencies
RUN npm install -g pnpm && pnpm install

# Copy the entire project
COPY . .

# Build the Vite app
RUN pnpm build

# Phase 2: Use Nginx to serve the built app
FROM nginx:stable-alpine

# Copy custom nginx.conf
COPY deploy/k8s-nginx.conf /etc/nginx/nginx.conf

# Copy the built files from the builder phase to the Nginx image
COPY --from=builder /app/dist /usr/share/nginx/html

# Expose port 80
EXPOSE 80

# Default command to start Nginx
CMD ["nginx", "-g", "daemon off;"]

