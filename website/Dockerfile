# The Typewriter website: build with Node, serve the finished files with nginx.
# Dokploy: Build type "Dockerfile", Build path "website", port 80.
FROM node:22-alpine AS build
WORKDIR /app
COPY package.json package-lock.json .npmrc ./
RUN npm ci
COPY . .
# SITE_URL (optional): the address of the site, for example https://typewriter.example.com. It makes the link preview picture a full address.
ARG SITE_URL=""
RUN SITE_URL="$SITE_URL" npm run build

FROM nginx:1.27-alpine
COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /app/dist /usr/share/nginx/html
EXPOSE 80
