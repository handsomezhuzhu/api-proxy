# 阶段一: 编译
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY main.go .

# 编译 Go 程序，使用 CGO_ENABLED=0 生成静态链接的可执行文件
RUN go build -ldflags "-s -w" -o api-proxy main.go

# 阶段二: 运行
FROM alpine:3.19
WORKDIR /app
# 从编译阶段复制可执行文件
COPY --from=builder /app/api-proxy .

# 暴露您的程序监听端口 (假设您在 main.go 中设置为 8080)
EXPOSE 7890

# 定义容器启动命令
CMD ["./api-proxy"]



