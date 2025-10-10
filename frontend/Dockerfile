# =========================================
# Build stage
# =========================================
FROM node:20-alpine AS build
WORKDIR /app

# Install deps (respect lockfile)
COPY package*.json ./
RUN npm ci

# Copy source and build
COPY . .
RUN npm run build

# =========================================
# Runtime stage
# =========================================
FROM nginx:alpine
# Nginx config with SPA fallback and basic caching
COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /app/dist /usr/share/nginx/html

EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
