FROM golang:1.20-alpine AS builder

WORKDIR /app
RUN apk add git && git clone https://github.com/bincooo/chatgpt-adapter.git -b v2.0.0 .
RUN go mod tidy && GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" -o linux-server -trimpath

FROM alpine:3.19.0
WORKDIR /app
COPY --from=builder /app/linux-server ./linux-server
COPY --from=builder /app/config.yaml ./config.yaml
RUN chmod +x linux-server

# 代理参数使用方法
# ENTRYPOINT ["sh","-c","./linux-server --proxies http://127.0.0.1:7890"]

# 无代理使用
ENTRYPOINT ["sh","-c","./linux-server"]