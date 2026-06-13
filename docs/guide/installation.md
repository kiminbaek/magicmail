# 安装部署

## 从源码编译

详见 [快速开始](/guide/getting-started)。

## 直接下载二进制

在 [GitHub Releases](https://github.com/magiccode1412/magicmail/releases) 下载对应平台的预编译二进制：

| 平台 | 文件 |
|------|------|
| Linux amd64 | `magicmail-linux-amd64` |
| Linux arm64 | `magicmail-linux-arm64` |
| macOS amd64 | `magicmail-darwin-amd64` |
| macOS arm64 | `magicmail-darwin-arm64` |
| Windows amd64 | `magicmail-windows-amd64.exe` |

下载后赋予执行权限并运行：

```bash
chmod +x magicmail-linux-amd64
./magicmail-linux-amd64
```

## Docker 部署

```bash
docker build -t magicmail .
docker run -d \
  -p 8080:8080 \
  -v ./data:/app/data \
  --name magicmail \
  magicmail
```

## systemd 服务（Linux）

项目内置了 systemd 服务配置文件 `server/magicmail.service`：

```bash
# 复制服务文件
sudo cp server/magicmail.service /etc/systemd/system/

# 根据实际路径修改 ExecStart
sudo systemctl daemon-reload
sudo systemctl enable magicmail
sudo systemctl start magicmail
```

## 反向代理配置

### Nginx

```nginx
server {
    listen 80;
    server_name mail.example.com;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

### Caddy

```
mail.example.com {
    reverse_proxy localhost:8080
}
```
