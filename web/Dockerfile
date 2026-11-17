
# Stage 1: Build React application
FROM node:22-alpine AS builder

WORKDIR /app

COPY package*.json ./
RUN npm ci

COPY . .
RUN npm run build


# Stage 2: Production Nginx server
FROM nginx:alpine

COPY --from=builder /app/dist /usr/share/nginx/html

COPY nginx/default.conf.template /etc/nginx/templates/default.conf.template

EXPOSE 8080

