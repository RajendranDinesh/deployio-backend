FROM debian:bookworm-slim

RUN apt-get update && \
    apt-get install -y \
    wget \
    curl \
    git \
    build-essential \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/* \
    && apt-get clean \
    && wget https://go.dev/dl/go1.22.6.linux-amd64.tar.gz \
    && tar -C /usr/local -xzf go1.22.6.linux-amd64.tar.gz \
    && rm go1.22.6.linux-amd64.tar.gz

ENV GOPATH=/go \
    PATH=/usr/local/go/bin:/go/bin:$PATH \
    NVM_DIR=/usr/local/nvm \
    NODE_VERSION=v18.20.4

WORKDIR /buildServer

COPY . .

RUN go mod download \
    && go build \
    && mkdir -p $NVM_DIR \
    && curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.40.0/install.sh | bash \
    && /bin/bash -c "source $NVM_DIR/nvm.sh && nvm install $NODE_VERSION && nvm use --delete-prefix $NODE_VERSION"

ENV NODE_PATH=$NVM_DIR/versions/node/$NODE_VERSION/lib/node_modules \
    PATH=$NVM_DIR/versions/node/$NODE_VERSION/bin:$PATH \
    TZ=Asia/Kolkata

CMD ["./buildServer"]
