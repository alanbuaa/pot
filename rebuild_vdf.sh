#!/bin/bash
# VDF 重新编译脚本
# 用途: 解决 GLIBC 版本不兼容问题

set -e  # 遇到错误立即退出

VDF_DIR="/home/ldc/workspace/pot/crypto/blockchain-crypto/vdf/utils"
VDF_BINARY="$VDF_DIR/vdf-linux"
TMP_DIR="/tmp/vdf-rebuild"

echo "========================================="
echo "VDF 重新编译脚本"
echo "========================================="
echo ""

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 检查当前问题
echo "1. 诊断当前问题..."
if ldd "$VDF_BINARY" 2>&1 | grep -q "not found"; then
    echo -e "${RED}✗ VDF 可执行文件存在 GLIBC 兼容性问题${NC}"
    ldd "$VDF_BINARY" 2>&1 | grep "not found" | head -3
    echo ""
else
    echo -e "${GREEN}✓ VDF 可执行文件依赖正常${NC}"
    echo "无需重新编译，脚本退出。"
    exit 0
fi

# 检查 Rust 是否安装
echo "2. 检查 Rust 工具链..."
if ! command -v rustc &> /dev/null; then
    echo -e "${YELLOW}! Rust 未安装，正在安装...${NC}"
    curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y
    source "$HOME/.cargo/env"
    echo -e "${GREEN}✓ Rust 安装完成${NC}"
else
    echo -e "${GREEN}✓ Rust 已安装: $(rustc --version)${NC}"
fi

# 检查依赖库
echo ""
echo "3. 检查依赖库..."
if ! dpkg -l | grep -q libgmp-dev; then
    echo -e "${YELLOW}! libgmp-dev 未安装，正在安装...${NC}"
    sudo apt-get update -qq
    sudo apt-get install -y libgmp-dev build-essential
    echo -e "${GREEN}✓ 依赖库安装完成${NC}"
else
    echo -e "${GREEN}✓ libgmp-dev 已安装${NC}"
fi

# 备份原文件
echo ""
echo "4. 备份原 VDF 可执行文件..."
if [ -f "$VDF_BINARY" ]; then
    cp "$VDF_BINARY" "${VDF_BINARY}.bak.$(date +%Y%m%d_%H%M%S)"
    echo -e "${GREEN}✓ 已备份到 ${VDF_BINARY}.bak.$(date +%Y%m%d_%H%M%S)${NC}"
fi

# 克隆或更新源码
echo ""
echo "5. 获取 VDF 源码..."
if [ -d "$TMP_DIR" ]; then
    echo "清理旧的临时目录..."
    rm -rf "$TMP_DIR"
fi

mkdir -p "$TMP_DIR"
cd "$TMP_DIR"

git clone --depth 1 https://github.com/poanetwork/vdf.git
cd vdf

echo -e "${GREEN}✓ 源码下载完成${NC}"

# 修复 Rust 兼容性问题
echo ""
echo "5.5. 修复 Rust 兼容性问题..."
# 修复 build.rs 中的宏问题
if [ -f "vdf/build.rs" ]; then
    sed -i 's/const_fmt!()/const_fmt!{}/' vdf/build.rs 2>/dev/null || true
    sed -i 's/"#\[allow(warnings)\]\\nconst {}: \[{}; {}\] = {:#?};\\n\\n";/"#[allow(warnings)]\\nconst {}: [{}; {}] = {:#?}\\n\\n"/' vdf/build.rs
    echo -e "${GREEN}✓ 已修复兼容性问题${NC}"
fi

# 编译
echo ""
echo "6. 编译 VDF (这可能需要几分钟)..."
source "$HOME/.cargo/env"

# 设置 Rust 版本以允许旧的宏语法
export RUSTFLAGS="-A semicolon_in_expressions_from_macros"
cargo build --release --bin vdf-cli

if [ ! -f "target/release/vdf-cli" ]; then
    echo -e "${RED}✗ 编译失败，找不到 vdf-cli${NC}"
    exit 1
fi

echo -e "${GREEN}✓ 编译完成${NC}"

# 安装新文件
echo ""
echo "7. 安装新的 VDF 可执行文件..."
cp target/release/vdf-cli "$VDF_BINARY"
chmod +x "$VDF_BINARY"
echo -e "${GREEN}✓ 安装完成${NC}"

# 验证
echo ""
echo "8. 验证新的 VDF 可执行文件..."
echo "   检查依赖:"
if ldd "$VDF_BINARY" 2>&1 | grep -q "not found"; then
    echo -e "${RED}✗ 仍然存在依赖问题${NC}"
    ldd "$VDF_BINARY"
    exit 1
else
    echo -e "${GREEN}✓ 所有依赖正常${NC}"
    ldd "$VDF_BINARY" | grep -E "libc.so|libgmp.so"
fi

echo ""
echo "   功能测试:"
TEST_OUTPUT=$("$VDF_BINARY" 6161 1000 2>&1)
if [ $? -eq 0 ] && [ -n "$TEST_OUTPUT" ]; then
    echo -e "${GREEN}✓ VDF 计算测试通过${NC}"
    echo "   输出: ${TEST_OUTPUT:0:64}..."
else
    echo -e "${RED}✗ VDF 计算测试失败${NC}"
    echo "   输出: $TEST_OUTPUT"
    exit 1
fi

# 清理
echo ""
echo "9. 清理临时文件..."
cd /
rm -rf "$TMP_DIR"
echo -e "${GREEN}✓ 清理完成${NC}"

# 完成
echo ""
echo "========================================="
echo -e "${GREEN}✓ VDF 重新编译完成！${NC}"
echo "========================================="
echo ""
echo "下一步操作:"
echo "  1. 重新编译 PoT 项目: cd /home/ldc/workspace/pot && make build"
echo "  2. 启动服务器测试:    make run_server"
echo "  3. 观察日志中的 VDF 计算结果"
echo ""
echo "如果出现问题，可以恢复备份:"
echo "  ls -la $VDF_DIR/*.bak.*"
echo ""
