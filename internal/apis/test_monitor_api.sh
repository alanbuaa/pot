#!/bin/bash

# POT可视化API测试脚本
# 用于测试所有监控API端点

API_BASE="http://localhost:18080/api"

echo "========================================="
echo "POT可视化监控API测试"
echo "========================================="
echo ""

# 颜色定义
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 测试函数
test_endpoint() {
    local endpoint=$1
    local name=$2
    
    echo -e "${YELLOW}测试: ${name}${NC}"
    echo "GET ${API_BASE}${endpoint}"
    
    response=$(curl -s -o /dev/null -w "%{http_code}" "${API_BASE}${endpoint}")
    
    if [ "$response" = "200" ]; then
        echo -e "${GREEN}✓ 成功 (HTTP $response)${NC}"
        # 显示响应内容（格式化JSON）
        curl -s "${API_BASE}${endpoint}" | python3 -m json.tool 2>/dev/null || curl -s "${API_BASE}${endpoint}"
    else
        echo -e "${RED}✗ 失败 (HTTP $response)${NC}"
    fi
    echo ""
    echo "-----------------------------------------"
    echo ""
}

# 检查服务器是否运行
echo "检查API服务器状态..."
if ! curl -s "${API_BASE}/system/overview" > /dev/null 2>&1; then
    echo -e "${RED}错误: API服务器未运行!${NC}"
    echo "请先启动服务器: make run_server"
    exit 1
fi
echo -e "${GREEN}✓ API服务器正在运行${NC}"
echo ""
echo "-----------------------------------------"
echo ""

# 测试所有端点
test_endpoint "/system/overview" "系统概览"
test_endpoint "/pot/status" "POT共识状态"
test_endpoint "/pot/vdf" "VDF计算状态"
test_endpoint "/committee/status" "委员会状态"
test_endpoint "/bci/status" "BCI激励状态"
test_endpoint "/mempool/status" "交易池状态"
test_endpoint "/network/topology" "网络拓扑"

echo "========================================="
echo "测试完成!"
echo "========================================="
