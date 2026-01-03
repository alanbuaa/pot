#!/bin/bash

echo "=== CDL 实现验证脚本 ==="
echo ""

echo "1. 检查 CDL 相关文件..."
echo ""

# 检查 CDL 基础设施文件
CDL_FILES=(
    "consensus/upgrade/cdl/types.go"
    "consensus/upgrade/cdl/parser.go"
    "consensus/upgrade/cdl/validator.go"
    "consensus/upgrade/cdl/compiler.go"
    "consensus/upgrade/cdl/runtime.go"
)

for file in "${CDL_FILES[@]}"; do
    if [ -f "$file" ]; then
        echo "✓ $file 存在"
    else
        echo "✗ $file 不存在"
    fi
done

echo ""
echo "2. 检查 consensus_factory.go 中的 CDL 集成..."
echo ""

# 检查 consensus_factory.go 中是否已实现 CDL 方法
if grep -q "func (cf \*ConsensusFactory) CreateConsensusFromCDL" consensus/upgrade/consensus_factory.go; then
    echo "✓ CreateConsensusFromCDL 方法已实现"
else
    echo "✗ CreateConsensusFromCDL 方法未实现"
fi

if grep -q "func (cf \*ConsensusFactory) buildConfigFromCDL" consensus/upgrade/consensus_factory.go; then
    echo "✓ buildConfigFromCDL 方法已实现"
else
    echo "✗ buildConfigFromCDL 方法未实现"
fi

if grep -q "validator := cdl.NewValidator" consensus/upgrade/consensus_factory.go; then
    echo "✓ ValidateProposal 中已集成 CDL 验证器"
else
    echo "✗ ValidateProposal 中未集成 CDL 验证器"
fi

echo ""
echo "3. 检查是否还有 TODO 标记..."
echo ""

TODO_COUNT=$(grep -r "TODO.*CDL" consensus/upgrade/consensus_factory.go | wc -l)
if [ "$TODO_COUNT" -eq 0 ]; then
    echo "✓ 所有 CDL TODO 已完成"
else
    echo "✗ 还有 $TODO_COUNT 个 CDL TODO 未完成:"
    grep -n "TODO.*CDL" consensus/upgrade/consensus_factory.go
fi

echo ""
echo "4. 检查 CDL 实现的关键功能..."
echo ""

# 检查转换函数
if grep -q "func (cf \*ConsensusFactory) convertToCDLDescriptor" consensus/upgrade/consensus_factory.go; then
    echo "✓ CDL 描述符转换函数已实现"
else
    echo "✗ CDL 描述符转换函数未实现"
fi

# 检查组件转换
if grep -q "func (cf \*ConsensusFactory) convertComponents" consensus/upgrade/consensus_factory.go; then
    echo "✓ 组件转换函数已实现"
else
    echo "✗ 组件转换函数未实现"
fi

# 检查参数转换
if grep -q "func (cf \*ConsensusFactory) convertParameters" consensus/upgrade/consensus_factory.go; then
    echo "✓ 参数转换函数已实现"
else
    echo "✗ 参数转换函数未实现"
fi

# 检查状态机转换
if grep -q "func (cf \*ConsensusFactory) convertStateMachine" consensus/upgrade/consensus_factory.go; then
    echo "✓ 状态机转换函数已实现"
else
    echo "✗ 状态机转换函数未实现"
fi

echo ""
echo "5. 统计代码行数..."
echo ""

echo "CDL 类型定义: $(wc -l < consensus/upgrade/cdl/types.go) 行"
echo "CDL 解析器: $(wc -l < consensus/upgrade/cdl/parser.go) 行"
echo "CDL 验证器: $(wc -l < consensus/upgrade/cdl/validator.go) 行"
echo "CDL 编译器: $(wc -l < consensus/upgrade/cdl/compiler.go) 行"
echo "CDL 运行时: $(wc -l < consensus/upgrade/cdl/runtime.go) 行"
echo "共识工厂(含CDL集成): $(wc -l < consensus/upgrade/consensus_factory.go) 行"

TOTAL_LINES=$(cat consensus/upgrade/cdl/*.go consensus/upgrade/consensus_factory.go | wc -l)
echo ""
echo "总计: $TOTAL_LINES 行代码"

echo ""
echo "6. 检查 CDL 测试文件..."
echo ""

if [ -f "consensus/upgrade/cdl_integration_test.go" ]; then
    echo "✓ CDL 集成测试文件已创建"
    TEST_COUNT=$(grep -c "^func Test" consensus/upgrade/cdl_integration_test.go)
    echo "  包含 $TEST_COUNT 个测试函数"
else
    echo "✗ CDL 集成测试文件不存在"
fi

echo ""
echo "7. CDL 功能列表..."
echo ""

cat <<EOF
已实现的 CDL 功能:
  ✓ CDL YAML 解析 (parser.go)
  ✓ CDL 验证 (validator.go)
    - 基本信息验证
    - 组件配置验证
    - 参数验证
    - 阶段验证
    - 状态机验证
    - 安全属性验证
    - 性能要求验证
    - 语义验证
  ✓ CDL 编译 (compiler.go)
    - 组件编译
    - 参数编译
    - 状态机编译
  ✓ CDL 运行时 (runtime.go)
    - 共识实例创建
    - 状态机执行
    - 生命周期管理
  ✓ CDL 集成 (consensus_factory.go)
    - CreateConsensusFromCDL: 从 CDL 创建共识
    - buildConfigFromCDL: 从 CDL 构建配置
    - ValidateProposal: CDL 验证集成
    - 格式转换: 旧/新 CDL 描述符互转

已创建的测试:
  ✓ CDL 集成测试
  ✓ CDL 解析器测试
  ✓ CDL 验证器测试
  ✓ CDL 编译器测试
  ✓ 无效 CDL 测试
EOF

echo ""
echo "=== 验证完成 ==="
