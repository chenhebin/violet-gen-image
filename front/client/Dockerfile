# syntax=docker/dockerfile:1.7

FROM node:22.16-alpine AS builder

WORKDIR /app
COPY package.json package-lock.json ./
RUN --mount=type=cache,target=/root/.npm \
    npm ci --ignore-scripts

COPY . .
ENV VITE_API_BASE_URL=/api \
    VITE_ENABLE_MOCKS=false
RUN npm run build \
    && rm -f /app/dist/mockServiceWorker.js

FROM nginxinc/nginx-unprivileged:1.27-alpine

COPY deploy/nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=builder /app/dist /usr/share/nginx/html

EXPOSE 8080
HEALTHCHECK --interval=15s --timeout=3s --start-period=5s --retries=3 \
  CMD wget -q -O - http://127.0.0.1:8080/healthz >/dev/null || exit 1
