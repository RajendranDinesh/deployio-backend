FROM golang:1.22

WORKDIR /

COPY . ./buildServer

WORKDIR /buildServer

RUN go mod download

RUN go build

ENV NVM_DIR /usr/local/nvm

RUN mkdir -p $NVM_DIR

RUN curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.40.0/install.sh | bash

ENV NODE_VERSION v18.20.4
RUN /bin/bash -c "source $NVM_DIR/nvm.sh && nvm install $NODE_VERSION && nvm use --delete-prefix $NODE_VERSION"
ENV NODE_PATH $NVM_DIR/versions/node/$NODE_VERSION/lib/node_modules
ENV PATH      $NVM_DIR/versions/node/$NODE_VERSION/bin:$PATH

ENV TZ Asia/Kolkata

CMD ["/buildServer/buildServer"]
