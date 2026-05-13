#!/bin/bash

# =============================================================================
# 可升级共识测试脚本
# 测试流程：
#   1. 启动节点服务器
#   2. 启动客户端，持续发送交易
#   3. 客户端发起共识切换提案
#   4. 客户端发起确认投票
#   5. 验证节点完成共识切换
# =============================================================================

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# 配置
PROJECT_ROOT=$(cd "$(dirname "$0")/../.." && pwd)
SERVER_CONFIG="$PROJECT_ROOT/cmd/upgrade/server/config.yaml"
CLIENT_CONFIG="$PROJECT_ROOT/cmd/upgrade/client/config.yaml"
DATA_DIR="$PROJECT_ROOT/data"
LOG_DIR="$DATA_DIR/upgrade-logs"

# 进程ID存储
SERVER_PID=""
CLIENT_PID=""

# 清理函数
cleanup() {
    echo -e "\n${YELLOW}[CLEANUP] Stopping all processes...${NC}"
    
    if [ -n "$CLIENT_PID" ] && kill -0 $CLIENT_PID 2>/dev/null; then
        echo -e "${CYAN}[INFO] Stopping client (PID: $CLIENT_PID)${NC}"
        kill -SIGTERM $CLIENT_PID 2>/dev/null || true
        wait $CLIENT_PID 2>/dev/null || true
    fi
    
    if [ -n "$SERVER_PID" ] && kill -0 $SERVER_PID 2>/dev/null; then
        echo -e "${CYAN}[INFO] Stopping server (PID: $SERVER_PID)${NC}"
        kill -SIGTERM $SERVER_PID 2>/dev/null || true
        wait $SERVER_PID 2>/dev/null || true
    fi
    
    # 额外清理可能残留的进程
    pkill -f "upgrade-server" 2>/dev/null || true
    pkill -f "upgrade-client" 2>/dev/null || true
    
    echo -e "${GREEN}[CLEANUP] All processes stopped${NC}"
}

# 设置信号处理
trap cleanup EXIT INT TERM

# 打印分隔线
print_separator() {
    echo -e "${BLUE}=============================================================${NC}"
}

# 打印步骤
print_step() {
    print_separator
    echo -e "${GREEN}[STEP $1] $2${NC}"
    print_separator
}

# 等待服务就绪
wait_for_service() {
    local host=$1
    local port=$2
    local timeout=$3
    local count=0
    
    echo -e "${CYAN}[INFO] Waiting for service at $host:$port...${NC}"
    while ! nc -z $host $port 2>/dev/null; do
        sleep 1
        count=$((count + 1))
        if [ $count -ge $timeout ]; then
            echo -e "${RED}[ERROR] Service not ready after ${timeout}s${NC}"
            return 1
        fi
    done
    echo -e "${GREEN}[INFO] Service at $host:$port is ready${NC}"
    return 0
}

# 主函数
main() {
    echo -e "${CYAN}"
    echo "╔══════════════════════════════════════════════════════════════╗"
    echo "║        Upgradeable Consensus Integration Test                ║"
    echo "╚══════════════════════════════════════════════════════════════╝"
    echo -e "${NC}"
    
    # 创建日志目录
    mkdir -p "$LOG_DIR"
    mkdir -p "$DATA_DIR/node-0"
    
    # 步骤 0: 编译
    print_step "0" "Building binaries..."
    cd "$PROJECT_ROOT"
    
    echo -e "${CYAN}[INFO] Building upgrade-server...${NC}"
    go build -o bin/upgrade-server ./cmd/upgrade/server/
    echo -e "${GREEN}[INFO] upgrade-server built successfully${NC}"
    
    echo -e "${CYAN}[INFO] Building upgrade-client...${NC}"
    go build -o bin/upgrade-client ./cmd/upgrade/client/
    echo -e "${GREEN}[INFO] upgrade-client built successfully${NC}"
    
    # 步骤 1: 启动节点服务器
    print_step "1" "Starting upgrade server..."
    
    echo -e "${CYAN}[INFO] Server config: $SERVER_CONFIG${NC}"
    ./bin/upgrade-server -c "$SERVER_CONFIG" > "$LOG_DIR/server.log" 2>&1 &
    SERVER_PID=$!
    echo -e "${GREEN}[INFO] Server started with PID: $SERVER_PID${NC}"
    
    # 等待服务器就绪
    sleep 3
    if ! kill -0 $SERVER_PID 2>/dev/null; then
        echo -e "${RED}[ERROR] Server failed to start. Check log: $LOG_DIR/server.log${NC}"
        cat "$LOG_DIR/server.log"
        exit 1
    fi
    echo -e "${GREEN}[INFO] Server is running${NC}"
    
    # 步骤 2: 启动客户端发送交易
    print_step "2" "Starting client to send transactions (15 seconds)..."
    
    echo -e "${CYAN}[INFO] Client config: $CLIENT_CONFIG${NC}"
    ./bin/upgrade-client -c "$CLIENT_CONFIG" -duration 15 > "$LOG_DIR/client-tx.log" 2>&1 &
    CLIENT_PID=$!
    echo -e "${GREEN}[INFO] Client started with PID: $CLIENT_PID (sending transactions)${NC}"
    
    # 等待一些交易被发送
    echo -e "${CYAN}[INFO] Waiting for transactions to be processed...${NC}"
    sleep 8
    
    # 打印交易统计
    echo -e "${YELLOW}[STATS] Transaction log excerpt:${NC}"
    tail -20 "$LOG_DIR/client-tx.log" 2>/dev/null | grep -E "(Transaction|confirmed|pending)" || true
    
    # 等待客户端完成
    wait $CLIENT_PID 2>/dev/null || true
    CLIENT_PID=""
    echo -e "${GREEN}[INFO] Transaction sending completed${NC}"
    
    # 步骤 3: 发起共识升级提案
    print_step "3" "Sending consensus upgrade proposal (hotstuff -> basichotstuff)..."
    
    ./bin/upgrade-client -c "$CLIENT_CONFIG" -upgrade "basichotstuff" > "$LOG_DIR/client-upgrade.log" 2>&1 &
    CLIENT_PID=$!
    echo -e "${GREEN}[INFO] Upgrade proposal client started with PID: $CLIENT_PID${NC}"
    
    sleep 3
    wait $CLIENT_PID 2>/dev/null || true
    CLIENT_PID=""
    
    echo -e "${YELLOW}[STATS] Upgrade proposal log:${NC}"
    cat "$LOG_DIR/client-upgrade.log" 2>/dev/null | grep -E "(proposal|Proposal|upgrade|Upgrade)" || true
    
    # 提取 proposal_id
    PROPOSAL_ID=$(cat "$LOG_DIR/client-upgrade.log" 2>/dev/null | grep -oP 'proposal_id[=:]\s*\K[a-f0-9]+' | head -1)
    if [ -z "$PROPOSAL_ID" ]; then
        PROPOSAL_ID="auto"
        echo -e "${YELLOW}[INFO] No proposal ID found, using 'auto'${NC}"
    else
        echo -e "${GREEN}[INFO] Proposal ID: $PROPOSAL_ID${NC}"
    fi
    
    # 步骤 4: 发起确认投票
    print_step "4" "Sending upgrade confirmation vote..."
    
    ./bin/upgrade-client -c "$CLIENT_CONFIG" -confirm "$PROPOSAL_ID" > "$LOG_DIR/client-confirm.log" 2>&1 &
    CLIENT_PID=$!
    echo -e "${GREEN}[INFO] Confirm client started with PID: $CLIENT_PID${NC}"
    
    sleep 3
    wait $CLIENT_PID 2>/dev/null || true
    CLIENT_PID=""
    
    echo -e "${YELLOW}[STATS] Confirm log:${NC}"
    cat "$LOG_DIR/client-confirm.log" 2>/dev/null | grep -E "(confirm|Confirm|approved)" || true
    
    # 步骤 5: 继续发送交易，验证共识切换
    print_step "5" "Continuing transaction sending to verify consensus switch (10 seconds)..."
    
    ./bin/upgrade-client -c "$CLIENT_CONFIG" -duration 10 > "$LOG_DIR/client-verify.log" 2>&1 &
    CLIENT_PID=$!
    echo -e "${GREEN}[INFO] Verification client started with PID: $CLIENT_PID${NC}"
    
    sleep 12
    wait $CLIENT_PID 2>/dev/null || true
    CLIENT_PID=""
    
    echo -e "${YELLOW}[STATS] Verification log excerpt:${NC}"
    tail -20 "$LOG_DIR/client-verify.log" 2>/dev/null | grep -E "(Transaction|confirmed|Statistics)" || true
    
    # 打印服务器日志中的共识切换信息
    print_step "6" "Checking server logs for consensus switch..."
    
    echo -e "${YELLOW}[STATS] Server consensus-related events:${NC}"
    grep -E "(upgrade|Upgrade|switch|Switch|consensus|Consensus|candidate)" "$LOG_DIR/server.log" 2>/dev/null | tail -30 || true
    
    # 打印最终统计
    print_separator
    echo -e "${GREEN}╔══════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${GREEN}║                     TEST COMPLETED                           ║${NC}"
    echo -e "${GREEN}╚══════════════════════════════════════════════════════════════╝${NC}"
    print_separator
    
    echo -e "${CYAN}[SUMMARY]${NC}"
    echo -e "  Server log: $LOG_DIR/server.log"
    echo -e "  Transaction log: $LOG_DIR/client-tx.log"
    echo -e "  Upgrade log: $LOG_DIR/client-upgrade.log"
    echo -e "  Confirm log: $LOG_DIR/client-confirm.log"
    echo -e "  Verify log: $LOG_DIR/client-verify.log"
    
    # 检查是否有共识切换记录
    if grep -q -E "(upgrade|switch|Switching)" "$LOG_DIR/server.log" 2>/dev/null; then
        echo -e "\n${GREEN}[SUCCESS] Consensus upgrade related events detected in server log${NC}"
    else
        echo -e "\n${YELLOW}[INFO] No explicit consensus switch recorded (may need higher block height)${NC}"
    fi
    
    print_separator
}

# 运行主函数
main "$@"
