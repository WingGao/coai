# Author: WingGao
# docker build -t registry.cn-shanghai.aliyuncs.com/mega_stg/chatnio:latest -f wing.Dockerfile ./
# docker push registry.cn-shanghai.aliyuncs.com/mega_stg/chatnio:latest

FROM m.daocloud.io/docker.io/library/golang:1.20-alpine AS backend

WORKDIR /backend
COPY . .

# Set go proxy to https://goproxy.cn (open for vps in China Mainland)
RUN go env -w GOPROXY=https://goproxy.cn,direct
ENV GOOS=linux GO111MODULE=on CGO_ENABLED=1

RUN echo "https://mirrors.aliyun.com/alpine/v3.19/main" > /etc/apk/repositories && \
    echo "https://mirrors.aliyun.com/alpine/v3.19/community" >> /etc/apk/repositories
# Install dependencies for cgo
RUN apk add --no-cache \
    gcc \
    musl-dev \
    g++ \
    make \
    linux-headers \
    libc-dev \
    pkgconfig \
    build-base \
    binutils-gold

# Build backend
RUN go install && \
    go build .

FROM m.daocloud.io/docker.io/library/node:18 AS frontend

WORKDIR /app
COPY ./app .
RUN npm config set registry https://registry.npmmirror.com
RUN npm install -g pnpm && \
    pnpm install
RUN pnpm run build && \
    rm -rf node_modules src


FROM m.daocloud.io/docker.io/library/alpine

RUN echo "https://mirrors.aliyun.com/alpine/v3.19/main" > /etc/apk/repositories && \
    echo "https://mirrors.aliyun.com/alpine/v3.19/community" >> /etc/apk/repositories
# Install dependencies
RUN apk upgrade --no-cache && \
    apk add --no-cache wget ca-certificates tzdata && \
    update-ca-certificates 2>/dev/null || true

# Set timezone
RUN echo "Asia/Shanghai" > /etc/timezone && \
    ln -sf /usr/share/zoneinfo/Asia/Shanghai /etc/localtime

WORKDIR /

# Copy dist
COPY --from=backend /backend/chat /chat
COPY --from=backend /backend/config.example.yaml /config.example.yaml
COPY --from=backend /backend/utils/templates /utils/templates
COPY --from=backend /backend/addition/article/template.docx /addition/article/template.docx
COPY --from=frontend /app/dist /app/dist

# Volumes
VOLUME ["/config", "/logs", "/storage"]

# Expose port
EXPOSE 8094

# Run application
CMD ["./chat"]
