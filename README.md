# WebDAV-115drive

![Go Version](https://img.shields.io/badge/go-1.24+-blue.svg)

> 一个 115 网盘 WebDAV 只读服务

## 🐳 Docker

```bash
docker run --rm -d \
  --name webdav-115drive \
  -p 8090:8090 \
  -e SERVER_USER=root \
  -e SERVER_PWD=123456 \
  -e DRIVE_UID=xxx \
  -e DRIVE_CID=xxx \
  -e DRIVE_SEID=xxx \
  -e DRIVE_KID=xxx \
  heartleo/webdav-115drive
```

## 🚀 编译安装

### 1. ⚒️ 安装

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
