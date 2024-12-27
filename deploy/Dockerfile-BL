FROM golang:1.23-alpine AS builder

WORKDIR /app
RUN apk add git make && git clone https://github.com/bincooo/chatgpt-adapter.git .
RUN make install
RUN make build-linux

FROM ubuntu:latest

WORKDIR /app
COPY --from=builder /app/bin/linux/server ./server

RUN apt update \
  && apt-get install -y curl unzip wget gnupg2

# 下载过盾文件
RUN curl -JLO https://raw.githubusercontent.com/bincooo/chatgpt-adapter/refs/heads/hel/bin.zip
RUN echo -e 'server:\n  port: 8080' > ./config.yaml

# Install google
RUN wget -q -O - https://dl.google.com/linux/linux_signing_key.pub | apt-key add - \
  && echo "deb [arch=amd64] http://dl.google.com/linux/chrome/deb/ stable main" >> /etc/apt/sources.list.d/google-chrome.list \
  && apt-get update \
  && apt-get install -y google-chrome-stable

# Install Edge
#RUN wget -q -O - https://packages.microsoft.com/keys/microsoft.asc | gpg --dearmor | tee /etc/apt/trusted.gpg.d/microsoft.gpg >/dev/null \
#    && echo "deb https://packages.microsoft.com/repos/edge stable main" >> /etc/apt/sources.list.d/microsoft-edge.list \
#    && apt-get update -qqy \
#    && apt-get -qqy --no-install-recommends install microsoft-edge-stable

RUN unzip bin.zip \
  && chmod +x server \
  && chmod +x bin/linux/helper

ENV ARG "--port 8080"
CMD ["./server ${ARG}"]
ENTRYPOINT ["sh", "-c"]
