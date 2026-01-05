# webdav-115drive

![Go Version](https://img.shields.io/badge/go-1.24+-blue.svg)
[![Docker Pulls](https://img.shields.io/docker/pulls/heartleo/webdav-115drive.svg)](https://hub.docker.com/r/heartleo/webdav-115drive)

> 一个 115 网盘 WebDAV 只读服务

## 🐳 Docker 运行

```bash
docker run --rm -d \
  --name webdav-115drive \
  -p 8090:8090 \
  -e SERVER_USER=user \
  -e SERVER_PWD=passwd \
  -e DRIVE_UID=xxx \
  -e DRIVE_CID=xxx \
  -e DRIVE_SEID=xxx \
  -e DRIVE_KID=xxx \
  heartleo/webdav-115drive
```

## 🐋 Docker Compose 运行

```bash
cat > docker-compose.yml <<EOF
services:
  webdav:
    container_name: webdav-115drive
    image: "heartleo/webdav-115drive:latest"
    ports:
      - "8090:8090"
    env_file:
      - .env
    restart: unless-stopped
EOF

cat > .env <<EOF
SERVER_USER=user
SERVER_PWD=passwd
DRIVE_UID=xxx
DRIVE_CID=xxx
DRIVE_SEID=xxx
DRIVE_KID=xxx
EOF

docker-compose up -d
```

## 🚀 编译运行

### 1. ⚒️ 编译

```bash
git clone https://github.com/heartleo/webdav-115drive.git
cd webdav-115drive
go build -o webdav-115drive .
```

### 2. ⚙️ 配置

**使用 `.env` 文件**

```bash
cp .env.example .env
```

**使用 `config.yaml`**

```bash
cp config.yaml.example config.yaml
```

### 3. ✈️ 运行

```bash
./webdav-115drive
```
