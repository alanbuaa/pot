
# POT 可升级共识框架

[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/go-1.20+-00ADD8.svg)](https://go.dev)

## 📖 项目简介

本项目是一个可升级共识框架的参考实现，用于研究和演示可升级共识的设计、共识组件之间的交互以及与执行器的集成。项目采用 Go 语言开发，包含共识核心、网络层、P2P 层、Protobuf 定义和示例命令。

### 核心特性

基于 POT 共识，支持共识热切换。当前已支持的共识算法包括：

- **HotStuff** - 支持 Basic、Event-Driven、Chained 三种模式
- **POW** - 工作量证明
- **POT** - Proof of Time 时间证明
- **Whirly** - 轻量级共识
- **POS** - 权益证明（开发中）

## 📁 项目结构

```
pot/
├── cmd/                    # 可执行程序入口
│   ├── server/            # 共识节点服务器
│   ├── executor/          # 交易执行器
│   ├── client/            # 客户端工具
│   └── genkey/            # 密钥生成工具
├── consensus/             # 共识算法实现
│   ├── hotstuff/         # HotStuff 共识
│   ├── pot/              # POT 共识
│   ├── pow/              # POW 共识
│   └── whirly/           # Whirly 共识
├── pkg/proto/            # Protobuf 定义和生成文件
├── p2p/                  # P2P 网络层
├── config/               # 配置文件
├── data/keys/            # 密钥文件目录
├── bin/                  # 编译输出目录
└── web/                  # 可视化前端
```

## 🔧 环境准备

在开始之前，请确保已安装以下工具：

- **Go 1.20+** - [安装指南](https://go.dev/doc/install)
- **Make** - 构建工具
- **Protocol Buffers** - protoc 编译器
- **Go Protobuf 插件** - protoc-gen-go 和 protoc-gen-go-grpc

### 安装 Protobuf 插件

```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

验证安装：

```bash
go version
protoc --version
which protoc-gen-go
which protoc-gen-go-grpc
```

## 🚀 快速开始

### 方法一：本地单节点开发

适用于本地开发和调试，运行单个共识节点。

#### 1. 构建项目

```bash
# 构建所有核心二进制文件
make build

# 或构建所有 cmd/ 下的程序
make build-cmds
```

#### 2. 生成密钥

```bash
# 生成密钥文件到 data/keys/ 目录
make run_genkey
```

#### 3. 配置节点

编辑 [config/config.yaml](config/config.yaml) 文件，确认配置：

```yaml
address_ip: "127.0.0.1"
address_start_port: "6060"
total: 1  # 单节点
executor:
  type: remote
  address: 127.0.0.1:9877
```

#### 4. 启动服务

**终端 1 - 启动执行器：**

```bash
make run_executor
```

**终端 2 - 启动共识节点：**

```bash
make run_server
```

#### 5. 访问可视化界面

打开浏览器访问：http://localhost:8088

#### 6. 运行测试

```bash
# 运行所有测试
make test

# 清理构建产物
make clean
```

### 方法二：本地多节点容器测试

适用于在本地模拟多节点集群环境，使用 Docker Compose 部署。

#### 1. 准备环境

```bash
# 确保已安装 Docker 和 Docker Compose
docker --version
docker-compose --version

# 生成密钥（如果还未生成）
make run_genkey
```

#### 2. 构建前端资源

```bash
# 编译 Web 可视化界面
make build_web
```

#### 3. 配置多节点

编辑 [config/config_docker.yaml](config/config_docker.yaml)，配置多节点：

```yaml
address_ip: "0.0.0.0"
total: 4  # 节点数量
executor:
  address: executor:9877  # 使用容器名称
consensus:
  pot:
    executorAddress: executor:9877
```

对于多节点部署，需要修改 [docker-compose.yml](docker-compose.yml) 添加更多节点：

```yaml
services:
  executor:
    # ... executor 配置
  
  server-1:
    # ... server 配置
  
  server-2:
    # ... 复制并修改端口
  
  server-3:
    # ... 复制并修改端口
  
  server-4:
    # ... 复制并修改端口
```

#### 4. 启动集群

```bash
# 一键构建并启动所有服务
make docker_compose_build

# 或分步执行
docker-compose build
docker-compose up -d
```

#### 5. 查看运行状态

```bash
# 查看所有容器状态
docker-compose ps

# 查看日志
make docker_compose_logs

# 查看特定服务日志
docker-compose logs -f server
```

#### 6. 访问服务

- **Web 界面**: http://localhost:8088
- **HTTP API**: http://localhost:8088/api/
- **Executor RPC**: localhost:9877
- **Server P2P**: localhost:6060
- **Server RPC**: localhost:7070

#### 7. 停止集群

```bash
# 停止所有服务
make docker_compose_down

# 或完全清理（包括数据卷）
docker-compose down -v
```

#### 故障排查

```bash
# 查看容器日志
docker-compose logs -f

# 进入容器调试
docker exec -it pot-server /bin/bash
docker exec -it pot-executor /bin/bash

# 重新构建（清除缓存）
docker-compose build --no-cache
docker-compose up --build -d
```

详细的 Docker 部署文档请参考：[docs/deploy/DOCKER.md](docs/deploy/DOCKER.md)

### 方法三：广域网集群测试（Ansible 部署）

适用于在多台物理服务器或云主机上部署分布式集群，测试真实网络环境下的性能。

#### 1. 准备服务器环境

确保所有目标服务器满足以下条件：

- **操作系统**: Ubuntu 20.04+ / CentOS 7+ / Debian 10+
- **SSH 访问**: 配置免密登录
- **必要软件**: Docker、Docker Compose
- **网络**: 服务器之间可以互相访问指定端口（6060, 7070, 8088, 9877）

#### 2. 安装 Ansible

在控制机上安装 Ansible：

```bash
# Ubuntu/Debian
sudo apt update
sudo apt install ansible -y

# macOS
brew install ansible

# 验证安装
ansible --version
```

#### 3. 配置 Ansible Inventory

创建 `ansible/inventory.ini` 文件，定义服务器清单：

```ini
[pot_servers]
node1 ansible_host=192.168.1.101 ansible_user=ubuntu node_id=0
node2 ansible_host=192.168.1.102 ansible_user=ubuntu node_id=1
node3 ansible_host=192.168.1.103 ansible_user=ubuntu node_id=2
node4 ansible_host=192.168.1.104 ansible_user=ubuntu node_id=3

[pot_executor]
executor ansible_host=192.168.1.105 ansible_user=ubuntu

[all:vars]
ansible_python_interpreter=/usr/bin/python3
```

#### 4. 创建 Ansible Playbook

创建 `ansible/deploy.yml` 部署脚本：

```yaml
---
- name: 部署 POT 共识集群
  hosts: all
  become: yes
  vars:
    project_dir: /opt/pot
    config_file: config_prod.yaml
    
  tasks:
    - name: 安装 Docker
      include_role:
        name: docker
    
    - name: 创建项目目录
      file:
        path: "{{ project_dir }}"
        state: directory
        mode: '0755'
    
    - name: 复制项目文件
      synchronize:
        src: ../
        dest: "{{ project_dir }}"
        delete: yes
        rsync_opts:
          - "--exclude=.git"
          - "--exclude=node_modules"
          - "--exclude=bin"
    
    - name: 生成密钥
      command: "{{ project_dir }}/bin/genkey -p {{ project_dir }}/data/keys/ -k 3 -l 4"
      args:
        creates: "{{ project_dir }}/data/keys/pub.key"

- name: 部署 Executor 节点
  hosts: pot_executor
  become: yes
  vars:
    project_dir: /opt/pot
  
  tasks:
    - name: 启动 Executor 容器
      docker_container:
        name: pot-executor
        image: pot-executor:latest
        state: started
        restart_policy: unless-stopped
        ports:
          - "9877:9877"
        volumes:
          - "{{ project_dir }}/config/{{ config_file }}:/app/config/config.yaml:ro"
          - "{{ project_dir }}/data/keys:/app/data/keys"

- name: 部署 Server 节点
  hosts: pot_servers
  become: yes
  vars:
    project_dir: /opt/pot
  
  tasks:
    - name: 生成节点配置
      template:
        src: config_template.yaml.j2
        dest: "{{ project_dir }}/config/node_{{ node_id }}.yaml"
    
    - name: 启动 Server 容器
      docker_container:
        name: "pot-server-{{ node_id }}"
        image: pot-server:latest
        state: started
        restart_policy: unless-stopped
        network_mode: host
        volumes:
          - "{{ project_dir }}/config/node_{{ node_id }}.yaml:/app/config/config.yaml:ro"
          - "{{ project_dir }}/data/keys:/app/data/keys"
          - "{{ project_dir }}/data/node-{{ node_id }}:/app/data"
```

#### 5. 配置模板

创建 `ansible/templates/config_template.yaml.j2`：

```yaml
address_ip: "{{ ansible_host }}"
address_start_port: "6060"
rpc_address_start_port: "7070"
http_server_address: "0.0.0.0:8088"
total: {{ groups['pot_servers'] | length }}
executor:
  type: remote
  address: "{{ hostvars[groups['pot_executor'][0]]['ansible_host'] }}:9877"
# ... 其他配置
```

#### 6. 执行部署

```bash
# 测试连接
ansible -i ansible/inventory.ini all -m ping

# 执行部署
ansible-playbook -i ansible/inventory.ini ansible/deploy.yml

# 或使用 tags 分步部署
ansible-playbook -i ansible/inventory.ini ansible/deploy.yml --tags "docker"
ansible-playbook -i ansible/inventory.ini ansible/deploy.yml --tags "deploy"
```

#### 7. 监控集群状态

```bash
# 查看所有节点状态
ansible -i ansible/inventory.ini pot_servers -a "docker ps"

# 查看日志
ansible -i ansible/inventory.ini pot_servers -a "docker logs pot-server-{{ node_id }}"

# 检查服务健康状态
ansible -i ansible/inventory.ini pot_servers -m uri -a "url=http://localhost:8088/health"
```

#### 8. 更新和维护

```bash
# 滚动更新
ansible-playbook -i ansible/inventory.ini ansible/update.yml --limit node1
ansible-playbook -i ansible/inventory.ini ansible/update.yml --limit node2
# ... 依次更新

# 停止集群
ansible-playbook -i ansible/inventory.ini ansible/stop.yml

# 清理数据
ansible -i ansible/inventory.ini all -a "rm -rf /opt/pot/data/node-*"
```

#### 性能测试

部署完成后，可以使用压测工具：

```bash
# 在本地机器执行压测
./bin/upgrade-cli benchmark \
  --endpoints http://node1:8088,http://node2:8088,http://node3:8088,http://node4:8088 \
  --tps 1000 \
  --duration 300s
```

#### 注意事项

- 确保防火墙开放必要端口：6060（P2P）、7070（RPC）、8088（HTTP）、9877（Executor）
- 生产环境建议使用 TLS 加密通信
- 定期备份 `data/keys/` 目录
- 监控磁盘空间，区块数据会持续增长
- 建议使用日志聚合工具（如 ELK）收集分析日志

## 🛠️ 开发指南

### Protobuf 代码生成

修改 `pkg/proto/*.proto` 文件后，重新生成 Go 代码：

```bash
make compile_proto
```

### 常用 Makefile 命令

| 命令 | 说明 |
|------|------|
| `make build` | 构建核心二进制文件 |
| `make build-cmds` | 构建所有 cmd/ 下的程序 |
| `make run_genkey` | 生成密钥文件 |
| `make test` | 运行测试 |
| `make compile_proto` | 生成 Protobuf 代码 |
| `make run_server` | 启动服务器 |
| `make run_executor` | 启动执行器 |
| `make build_web` | 编译前端资源 |
| `make docker_build_images` | 构建 Docker 镜像 |
| `make docker_compose_build` | Docker Compose 一键部署 |
| `make clean` | 清理构建产物 |

### 调试

使用 VS Code 调试：

1. 在 `.vscode/launch.json` 中配置调试项
2. 在代码中设置断点
3. 按 F5 启动调试

## 🔍 故障排查

### 常见问题

**1. `make: go: No such file or directory`**

解决：安装 Go 并确保在 PATH 中

```bash
go version
```

**2. Protobuf 生成失败**

解决：检查 protoc 和插件是否正确安装

```bash
protoc --version
which protoc-gen-go
which protoc-gen-go-grpc
```

**3. 运行时找不到密钥**

解决：生成密钥文件

```bash
make run_genkey
ls data/keys/
```

**4. Docker 容器无法通信**

解决：检查网络配置和容器名称

```bash
docker network ls
docker network inspect pot-network
```

**5. 端口被占用**

解决：修改配置文件中的端口或停止占用端口的进程

```bash
lsof -i :8088
```

## 📚 相关文档

- [API 文档](web/API.md)
- [Docker 部署指南](docs/deploy/DOCKER.md)
- [可视化快速上手](web/QUICKSTART.md)
- [开发指南](web/DEVELOPMENT.md)
- [升级计划](docs/upgrade_plan/)
- [模块文档](docs/modules/)

## 📄 许可证

本项目采用 Apache 2.0 许可证，详见 [LICENSE](LICENSE) 文件。

## 🙏 致谢

感谢所有贡献者以及提供工具和参考实现的开源项目。

---

如需帮助，请提交 Issue 或查看文档。

