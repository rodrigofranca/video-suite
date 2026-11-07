# Dashboard image: a static build served by nginx.
# Build context: the repository root.

FROM node:22.23.4-alpine AS build
WORKDIR /app
COPY apps/dashboard/package.json apps/dashboard/package-lock.json ./
RUN npm ci
COPY apps/dashboard ./
# The API URL the browser calls; it is written into the build.
ARG PUBLIC_API_URL=http://localhost:8080
ENV PUBLIC_API_URL=$PUBLIC_API_URL
RUN npm run build

FROM nginx:1.30.6-alpine
COPY infra/docker/dashboard.nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /app/build /usr/share/nginx/html
