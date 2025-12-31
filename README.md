# webdav-115drive

![Go Version](https://img.shields.io/badge/go-1.24+-blue.svg)

> A WebDAV read-only server for 115 Drive

## 🐳 Docker Run

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

## 🚀 Build & Install

### 1. ⚒️ Install

```bash
git clone https://github.com/heartleo/webdav-115drive.git
cd webdav-115drive
go build -o webdav-115drive .
```

### 2. ⚙️ Configuration

**Use `.env` files**

```bash
cp .env.example .env
```

**Use `config.yaml`**

```bash
cp config.yaml.example config.yaml
```

### 3. ✈️ Run

```bash
./webdav-115drive
```
