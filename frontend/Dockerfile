FROM oven/bun:1.4.0-alpine AS build
ARG VITE_DEMO_LOGIN_ENABLED=false
ENV VITE_DEMO_LOGIN_ENABLED=$VITE_DEMO_LOGIN_ENABLED
WORKDIR /src
COPY package.json bun.lock ./
RUN bun install --frozen-lockfile
COPY . .
RUN bun run build

FROM caddy:2.10-alpine
COPY Caddyfile /etc/caddy/Caddyfile
COPY --from=build /src/dist /srv
EXPOSE 8080
