FROM golang:1.22

WORKDIR /buildServer

COPY . .

RUN go mod download && go build

ENV NVM_DIR=/usr/local/nvm \
    NODE_VERSION=v18.20.4 

RUN mkdir -p $NVM_DIR && \
    curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.40.0/install.sh | bash && \
    /bin/bash -c "source $NVM_DIR/nvm.sh && nvm install $NODE_VERSION && nvm use --delete-prefix $NODE_VERSION"

ENV NODE_PATH=$NVM_DIR/versions/node/$NODE_VERSION/lib/node_modules \
    PATH=$NVM_DIR/versions/node/$NODE_VERSION/bin:$PATH \
    TZ=Asia/Kolkata

CMD ["./buildServer"]
