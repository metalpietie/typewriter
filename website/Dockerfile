# The Typewriter website: build with Node, serve the finished files with nginx.
# Dokploy: Build type "Dockerfile", Build path "website", port 80.
FROM node:22-alpine AS build
WORKDIR /app
COPY package.json package-lock.json .npmrc ./
RUN npm ci
COPY . .
# Build-time arguments, all optional:
#   SITE_URL          the address of the site, for example https://typewriter.example.com (full address for link previews,
#                     and Umami counts visits only on this domain)
#   PUBLIC_UMAMI_SRC  the Umami script, for example https://umami.example.com/script.js
#   PUBLIC_UMAMI_ID   the website ID from Umami (Settings > Websites)
# Without the two Umami values the site has no analytics at all.
ARG SITE_URL=""
ARG PUBLIC_UMAMI_SRC=""
ARG PUBLIC_UMAMI_ID=""
RUN SITE_URL="$SITE_URL" PUBLIC_UMAMI_SRC="$PUBLIC_UMAMI_SRC" PUBLIC_UMAMI_ID="$PUBLIC_UMAMI_ID" npm run build

FROM nginx:1.27-alpine
ARG PUBLIC_UMAMI_SRC=""
COPY nginx.conf /etc/nginx/conf.d/default.conf
# the security rule (Content-Security-Policy) lets the page load the Umami script and send to it, and nothing else
RUN origin="$(echo "$PUBLIC_UMAMI_SRC" | sed -E 's#^(https?://[^/]+).*#\1#')" \
 && sed -i "s# __UMAMI__#${origin:+ $origin}#g" /etc/nginx/conf.d/default.conf
COPY --from=build /app/dist /usr/share/nginx/html
EXPOSE 80
