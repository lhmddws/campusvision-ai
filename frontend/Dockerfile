FROM node:20-alpine AS build
WORKDIR /app
ARG NPM_REGISTRY=https://registry.npmjs.org
COPY package.json pnpm-lock.yaml* package-lock.json* ./
RUN corepack enable && corepack prepare pnpm@latest --activate
RUN pnpm install --registry=$NPM_REGISTRY
COPY . .
RUN pnpm build:prod

FROM nginx:1.27-alpine
COPY --from=build /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
HEALTHCHECK --interval=30s --timeout=3s CMD wget -qO- http://localhost/ || exit 1
