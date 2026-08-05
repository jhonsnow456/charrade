FROM golang:1.26-alpine AS backend-builder
WORKDIR /app
COPY apps/backend/go.mod apps/backend/go.sum ./
RUN go mod download
COPY apps/backend/ ./
RUN CGO_ENABLED=0 go build -o /server ./cmd/server

FROM node:20-alpine AS web-builder
WORKDIR /app
COPY package.json pnpm-lock.yaml pnpm-workspace.yaml turbo.json ./
COPY apps/web/package.json ./apps/web/
RUN corepack enable && pnpm install --frozen-lockfile
COPY apps/web/ ./apps/web/
RUN pnpm --filter=web build

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=backend-builder /server /server
COPY --from=web-builder /app/apps/web/dist /dist
EXPOSE 8080
CMD ["/server"]