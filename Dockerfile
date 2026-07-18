# Build stage - Go backend
FROM golang:alpine AS backend-builder
WORKDIR /app
ENV GOTOOLCHAIN=auto
RUN apk add --no-cache gcc musl-dev
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 go build -o woopen ./cmd/server

# Build stage - Vue frontend
FROM node:20-alpine AS frontend-builder
WORKDIR /app
COPY web/package*.json ./
# 删除锁文件按容器平台重装，规避 npm optionalDependencies bug(npm/cli#4828)
# 否则 Alpine 下缺 @rollup/rollup-linux-*-musl 导致 vite build 失败
RUN rm -f package-lock.json && npm install
COPY web/ ./
RUN npm run build

# Final stage
FROM alpine:3.19
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app

# Copy backend binary
COPY --from=backend-builder /app/woopen .
# Copy frontend dist
COPY --from=frontend-builder /app/dist ./web/dist

# Create data directory
RUN mkdir -p /data

EXPOSE 10010

ENV WOOPEN_DATA_DIR=/data
ENV WOOPEN_PORT=10010

CMD ["./woopen"]
