#!/bin/bash

# POT 可视化前端项目安装脚本

echo "================================================"
echo "POT 共识可视化前端项目 - 依赖安装"
echo "================================================"

# 检查 Node.js 版本
if ! command -v node &> /dev/null; then
    echo "错误: 未安装 Node.js"
    echo "请先安装 Node.js (推荐版本 >= 16.x)"
    exit 1
fi

NODE_VERSION=$(node -v)
echo "✓ Node.js 版本: $NODE_VERSION"

# 检查 npm
if ! command -v npm &> /dev/null; then
    echo "错误: 未安装 npm"
    exit 1
fi

NPM_VERSION=$(npm -v)
echo "✓ npm 版本: $NPM_VERSION"

echo ""
echo "开始安装依赖..."
echo ""

# 安装依赖
npm install

if [ $? -eq 0 ]; then
    echo ""
    echo "================================================"
    echo "✓ 依赖安装成功！"
    echo "================================================"
    echo ""
    echo "下一步操作："
    echo "1. 启动开发服务器: npm run dev"
    echo "2. 构建生产版本: npm run build"
    echo "3. 预览构建结果: npm run preview"
    echo ""
    echo "开发服务器将运行在: http://localhost:3000"
    echo "API 代理目标: http://localhost:10000/api (node0)"
    echo ""
else
    echo ""
    echo "================================================"
    echo "✗ 依赖安装失败"
    echo "================================================"
    echo ""
    echo "请检查网络连接或尝试以下操作："
    echo "1. 清除 npm 缓存: npm cache clean --force"
    echo "2. 删除 node_modules: rm -rf node_modules"
    echo "3. 重新安装: npm install"
    echo ""
    exit 1
fi
