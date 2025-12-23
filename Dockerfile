# 后端构建阶段
FROM golang:1.23-alpine AS backend-builder
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

# 创建数据目录
RUN mkdir -p /app/data/backups

CMD ["./tv-tracker"]
