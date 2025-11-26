# Docker 部署指南

本项目提供了两个独立的 Docker 镜像：`pot-executor` 和 `pot-server`。

## 镜像说明

### pot-executor
- **包含二进制**: `genkey`, `executor`
- **暴露端口**: 9877 (Executor RPC)
- **主要功能**: 执行交易和生成区块

### pot-server
- **包含二进制**: `genkey`, `server`
- **包含资源**: Web 前端静态文件 (`web/dist/`)
- **暴露端口**: 
  - 6060 (P2P)
  - 7070 (RPC)
  - 8088 (HTTP API + Web UI)
- **主要功能**: 共识节点服务器和可视化界面

## 快速启动

### 方式一：使用 docker-compose (推荐)

1. **准备配置文件和密钥**
   ```bash
   # 生成密钥 (如果还没有)
   make genkey
   
   # 确保配置文件存在
   ls config/config.yaml
   ```

2. **构建并启动所有服务**
   ```bash
   # 构建镜像并启动
   make docker_compose_build
   
   # 或者分步操作
   docker-compose build
   docker-compose up -d
   ```

3. **查看日志**
   ```bash
   make docker_compose_logs
   # 或
   docker-compose logs -f
   ```

4. **停止服务**
   ```bash
   make docker_compose_down
   # 或
   docker-compose down
   ```

### 方式二：手动构建和运行

1. **构建前端**
   ```bash
   cd web
   npm install
   npm run build
   cd ..
   ```

2. **构建 Docker 镜像**
   ```bash
   # 构建 executor 镜像
   make docker_build_executor
   
   # 构建 server 镜像
   make docker_build_server
   
   # 或一次性构建两个镜像
   make docker_build_images
   ```

3. **运行容器**
   ```bash
   # 创建网络
   docker network create pot-network
   
   # 运行 executor
   docker run -d \
     --name pot-executor \
     --network pot-network \
     -v $(pwd)/config/config.yaml:/app/config/config.yaml:ro \
     -v $(pwd)/data/keys:/app/data/keys \
     -p 9877:9877 \
     pot-executor:latest
   
   # 运行 server
   docker run -d \
     --name pot-server \
     --network pot-network \
     -v $(pwd)/config/config.yaml:/app/config/config.yaml:ro \
     -v $(pwd)/data/keys:/app/data/keys \
     -v $(pwd)/data:/app/data \
     -p 6060:6060 \
     -p 7070:7070 \
     -p 8088:8088 \
     pot-server:latest
   ```

## 配置说明

### 配置文件挂载
- 配置文件通过 volume 从宿主机挂载到容器
- 默认路径: `./config/config_docker.yaml` -> `/app/config/config.yaml`
- 修改配置后需要重启容器生效

### 数据持久化
- `data/keys/`: 密钥文件目录，在容器间共享
- `data/`: 服务器数据目录 (仅 server 需要)

### 网络配置
配置文件中的地址需要注意：
- **executor** 服务在容器内监听 `0.0.0.0:9877`
- **server** 配置中 `executorAddress` 应该使用容器名称: `executor:9877`

示例配置修改 (`config/config.yaml`):
```yaml
consensus:
  pot:
    executorAddress: executor:9877  # 使用容器名称
    bciRpcAddress: 127.0.0.1:9866
```

## 访问服务

启动后可以通过以下地址访问：

- **Web 可视化界面**: http://localhost:8088
- **HTTP API**: http://localhost:8088/api/
- **Executor RPC**: localhost:9877
- **Server P2P**: localhost:6060
- **Server RPC**: localhost:7070

## 故障排查

### 查看容器日志
```bash
# 查看所有服务日志
docker-compose logs -f

# 查看特定服务日志
docker-compose logs -f server
docker-compose logs -f executor
```

### 进入容器调试
```bash
# 进入 server 容器
docker exec -it pot-server /bin/bash

# 进入 executor 容器
docker exec -it pot-executor /bin/bash
```

### 重新构建
```bash
# 强制重新构建
docker-compose build --no-cache

# 重新构建并启动
docker-compose up --build -d
```

## Makefile 命令汇总

```bash
# Docker Compose 操作
make docker_compose_build     # 构建并启动服务
make docker_compose_up         # 启动服务
make docker_compose_down       # 停止服务
make docker_compose_logs       # 查看日志

# 镜像构建
make docker_build_executor     # 构建 executor 镜像
make docker_build_server       # 构建 server 镜像
make docker_build_images       # 构建所有镜像

# 前端构建
make build_web                 # 编译前端静态文件
```

## 注意事项

1. **首次运行前**必须先生成密钥: `make genkey`
2. 确保 `web/dist/` 目录存在且包含编译后的前端文件
3. 配置文件中的网络地址需要适配容器环境
4. 数据目录 `data/` 会被持久化，删除需谨慎
