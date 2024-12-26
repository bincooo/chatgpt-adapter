FROM golang:1.23-alpine AS builder

WORKDIR /app
RUN apk add git make && git clone https://github.com/bincooo/chatgpt-adapter.git .
RUN make install
RUN make build-linux

FROM alpine:3.19.0
WORKDIR /app
COPY --from=builder /app/bin/linux/server ./server
RUN echo -e 'server:\n  port: 8080' > ./config.yaml
RUN chmod +x server

ENV ARG "--port 8080"
CMD ["./server ${ARG}"]
ENTRYPOINT ["sh", "-c"]