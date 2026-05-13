#!/bin/bash

# POT 可视化前端快速启动脚本

echo "================================================"
echo "POT 共识可视化前端 - 快速启动"
echo "================================================"
echo ""

# 检查是否已安装依赖
if [ ! -d "node_modules" ]; then
    echo "⚠️  未检测到 node_modules 目录"
    echo "正在运行安装脚本..."
    echo ""
    ./install.sh
    
    if [ $? -ne 0 ]; then
        echo ""
        echo "依赖安装失败，无法启动开发服务器"
        exit 1
    fi
fi

echo ""
echo "================================================"
echo "启动开发服务器..."
echo "================================================"
echo ""
echo "前端地址: http://localhost:3000"
echo "API 代理: http://localhost:10000/api (node0)"
echo ""
echo "请确保 node0 后端 API 已在 10000 端口运行"
echo ""
echo "按 Ctrl+C 停止服务器"
echo "================================================"
echo ""

npm run dev
