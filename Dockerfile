# 前端构建阶段
FROM node:20-alpine AS frontend-builder
WORKDIR /app/web
COPY web/package*.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

# 后端构建阶段
FROM golang:1.21-alpine AS backend-builder
RUN apk add --no-cache gcc musl-dev sqlite-dev
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -a -ldflags '-linkmode external -extldflags "-static"' -o tv-tracker ./cmd/server

# 运行阶段
FROM alpine:latest
RUN apk add --no-cache ca-certificates tzdata sqlite
ENV TZ=Asia/Shanghai
WORKDIR /app
COPY --from=backend-builder /app/tv-tracker .
COPY --from=frontend-builder /app/web/dist ./web/dist

# 创建数据目录
RUN mkdir -p /app/data

EXPOSE 8080
CMD ["./tv-tracker"]
