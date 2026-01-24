#!/bin/bash

# POT 区块链节点管理脚本
# 支持多命名空间、多节点、多执行器的初始化和管理

set -e

# ==================== 配置常量 ====================
WORKSPACE_DIR="deploy"
DEFAULT_NAMESPACE="test"

# 默认端口配置 (基础端口,每个节点递增)
BASE_P2P_PORT=7000
BASE_RPC_PORT=8000
BASE_BCI_RPC_PORT=9000
BASE_API_PORT=10000
BASE_EXECUTOR_PORT=12000
BASE_POT_EXECUTOR_PORT=13000

# 模板文件路径
SERVER_TEMPLATE="deploy/server-template.yaml"

# ==================== 颜色输出 ====================
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# ==================== 工具函数 ====================
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_step() {
    echo -e "${BLUE}[STEP]${NC} $1"
}

# 检查命令是否存在
check_command() {
    if ! command -v $1 &> /dev/null; then
        log_error "命令 '$1' 未找到,请先安装"
        exit 1
    fi
}

# 检查模板文件是否存在
check_templates() {
    if [[ ! -f "$SERVER_TEMPLATE" ]]; then
        log_error "服务器模板文件 '$SERVER_TEMPLATE' 不存在"
        exit 1
    fi
}

# 获取命名空间目录
get_namespace_dir() {
    local namespace=$1
    echo "$WORKSPACE_DIR/$namespace"
}

# 检查命名空间是否存在
check_namespace_exists() {
    local namespace=$1
    local ns_dir=$(get_namespace_dir "$namespace")
    if [[ ! -d "$ns_dir" ]]; then
        log_error "命名空间 '$namespace' 不存在"
        exit 1
    fi
}

# ==================== 初始化功能 ====================
init_namespace() {
    local namespace=$1
    local num_nodes=$2
    local num_executors=$3

    # 参数验证
    if [[ -z "$namespace" ]]; then
        log_error "命名空间名称不能为空"
        exit 1
    fi

    if [[ ! "$num_nodes" =~ ^[0-9]+$ ]] || [[ "$num_nodes" -lt 1 ]]; then
        log_error "节点数量必须是大于0的整数"
        exit 1
    fi

    if [[ ! "$num_executors" =~ ^[0-9]+$ ]] || [[ "$num_executors" -lt 0 ]]; then
        log_error "执行器数量必须是非负整数"
        exit 1
    fi

    # 执行器数量不能超过节点数量
    if [[ "$num_executors" -gt "$num_nodes" ]]; then
        log_error "执行器数量($num_executors)不能超过节点数量($num_nodes)"
        exit 1
    fi

    local ns_dir=$(get_namespace_dir "$namespace")

    # 检查命名空间是否已存在
    if [[ -d "$ns_dir" ]]; then
        log_warn "命名空间 '$namespace' 已存在"
        read -p "是否覆盖? (y/N): " confirm
        if [[ ! "$confirm" =~ ^[Yy]$ ]]; then
            log_info "操作已取消"
            exit 0
        fi
        log_step "清理旧的命名空间..."
        rm -rf "$ns_dir"
    fi

    log_step "创建命名空间: $namespace (节点数: $num_nodes, 执行器数: $num_executors)"

    # 创建目录结构
    mkdir -p "$ns_dir/configs/servers"
    mkdir -p "$ns_dir/keys"
    mkdir -p "$ns_dir/data"
    mkdir -p "$ns_dir/logs"

    # 生成节点配置文件
    log_step "生成节点配置文件..."
    for ((i=0; i<$num_nodes; i++)); do
        generate_server_config "$ns_dir" "$i" "$num_nodes"
    done

    # 生成密钥
    log_step "生成节点密钥..."
    generate_keys "$ns_dir" "$num_nodes"

    log_info "命名空间 '$namespace' 初始化完成!"
    log_info "配置文件目录: $ns_dir/configs"
    log_info "密钥目录: $ns_dir/keys"
    log_info "数据目录: $ns_dir/data"
    log_info "日志目录: $ns_dir/logs"
    log_info "节点数: $num_nodes, 执行器数: $num_executors"
    
    # 保存执行器数量到配置文件，供启动时使用
    echo "$num_executors" > "$ns_dir/.executor_count"
}

# 生成服务器配置文件
generate_server_config() {
    local ns_dir=$1
    local node_id=$2
    local total_nodes=$3

    local p2p_port=$((BASE_P2P_PORT + node_id))
    local rpc_port=$((BASE_RPC_PORT + node_id))
    local bci_rpc_port=$((BASE_BCI_RPC_PORT + node_id))
    local api_port=$((BASE_API_PORT + node_id))
    local executor_port=$((BASE_EXECUTOR_PORT + node_id))
    local pot_executor_port=$((BASE_POT_EXECUTOR_PORT + node_id))

    local config_file="$ns_dir/configs/servers/node-$node_id.yaml"
    local private_key_path="$ns_dir/keys/node-$node_id-priv.key"
    local public_key_path="$ns_dir/keys/node-pub.key"
    local data_dir="$ns_dir/data/node-$node_id"
    local log_file="$ns_dir/logs/node-$node_id-%s.log"

    # 复制模板并替换变量
    cat "$SERVER_TEMPLATE" | \
        sed "s|{{NODE_ID}}|$node_id|g" | \
        sed "s|{{ADDRESS_PORT}}|$p2p_port|g" | \
        sed "s|{{RPC_PORT}}|$rpc_port|g" | \
        sed "s|{{BCI_RPC_PORT}}|$bci_rpc_port|g" | \
        sed "s|{{API_PORT}}|$api_port|g" | \
        sed "s|{{PRIVATE_KEY}}|$private_key_path|g" | \
        sed "s|{{PUBLIC_KEY}}|$public_key_path|g" | \
        sed "s|{{DATADIR}}|$data_dir|g" | \
        sed "s|{{LOG_FILE}}|$log_file|g" | \
        sed "s|{{TOTAL_REPLICAS}}|$total_nodes|g" | \
        sed "s|{{EXECUTOR_PORT}}|$executor_port|g" | \
        sed "s|{{POT_EXECUTOR_PORT}}|$pot_executor_port|g" \
        > "$config_file"

    log_info "生成节点配置: node-$node_id (P2P:$p2p_port, RPC:$rpc_port, BCI:$bci_rpc_port, API:$api_port, Executor:$executor_port, PotExecutor:$pot_executor_port)"
}


# 生成密钥
generate_keys() {
    local ns_dir=$1
    local num_nodes=$2
    
    log_info "使用 genkey 工具生成 $num_nodes 个节点密钥..."
    
    # 确保 bin/genkey 存在
    if [[ ! -f "bin/genkey" ]]; then
        log_warn "genkey 二进制文件不存在,尝试构建..."
        make build-genkey
    fi
    
    # 计算阈值签名参数
    # 密钥生成库要求:
    # 1. l (总密钥数) 必须 > 1
    # 2. k (签名阈值) 必须在 (l/2)+1 到 l 之间
    # 因此,当节点数 < 2 时,使用最小要求 l=2
    
    local threshold_k
    local threshold_l
    
    if [[ $num_nodes -lt 2 ]]; then
        # 最小密钥生成要求: l=2, k=2
        threshold_l=2
        threshold_k=2
        log_warn "节点数少于2,使用最小密钥生成参数 (k=$threshold_k, l=$threshold_l)"
    else
        # 正常计算: k = 2f+1 (f为拜占庭容错参数)
        threshold_k=$((num_nodes * 2 / 3 + 1))
        threshold_l=$num_nodes
    fi
    
    # 直接生成密钥到命名空间的 keys 目录
    log_info "生成阈值签名密钥 (k=$threshold_k, l=$threshold_l)..."
    ./bin/genkey -p "$ns_dir/keys" -k "$threshold_k" -l "$threshold_l" || {
        log_error "密钥生成失败"
        exit 1
    }
    
    # genkey 生成的私钥文件名是 node-{i}.key, 需要重命名为 node-{i}-priv.key
    # 只重命名实际需要的节点密钥
    for ((i=0; i<$num_nodes; i++)); do
        if [[ -f "$ns_dir/keys/node-$i.key" ]]; then
            mv "$ns_dir/keys/node-$i.key" "$ns_dir/keys/node-$i-priv.key"
            log_info "节点 $i 密钥已生成: $ns_dir/keys/node-$i-priv.key"
        else
            log_warn "未找到节点 $i 的密钥文件"
        fi
    done
    
    # 清理多余的密钥文件 (当 threshold_l > num_nodes 时)
    for ((i=$num_nodes; i<$threshold_l; i++)); do
        if [[ -f "$ns_dir/keys/node-$i.key" ]]; then
            rm -f "$ns_dir/keys/node-$i.key"
            log_info "清理多余的密钥文件: node-$i.key"
        fi
    done
    
    # 验证公钥文件是否存在
    if [[ -f "$ns_dir/keys/node-pub.key" ]]; then
        log_info "公钥文件已生成: $ns_dir/keys/node-pub.key"
    else
        log_error "公钥文件生成失败"
        exit 1
    fi
    
    log_info "密钥生成完成! 共生成 $num_nodes 个节点密钥"
}

# ==================== 节点操作功能 ====================
start_servers() {
    local namespace=$1
    local node_id=${2:-"all"}
    local follow_log=${3:-false}

    check_namespace_exists "$namespace"
    
    local ns_dir=$(get_namespace_dir "$namespace")
    local pids_file="$ns_dir/.server_pids"

    # 确保 bin/server 存在
    if [[ ! -f "bin/server" ]]; then
        log_warn "server 二进制文件不存在,尝试构建..."
        make build-server
    fi

    if [[ "$node_id" == "all" ]]; then
        log_step "启动命名空间 '$namespace' 的所有服务器节点..."
        
        # 清空 PID 文件
        > "$pids_file"
        
        # 启动所有节点
        local configs=("$ns_dir/configs/servers"/node-*.yaml)
        for config in "${configs[@]}"; do
            if [[ -f "$config" ]]; then
                local node_num=$(basename "$config" .yaml | sed 's/node-//')
                start_single_server "$namespace" "$node_num" "$pids_file"
            fi
        done
        
        log_info "所有服务器节点已启动"
        
        if [[ "$follow_log" == true ]]; then
            log_info "查看日志: tail -f $ns_dir/logs/node-*.log"
        fi
    else
        log_step "启动命名空间 '$namespace' 的服务器节点 $node_id..."
        start_single_server "$namespace" "$node_id" "$pids_file"
        
        if [[ "$follow_log" == true ]]; then
            log_info "跟踪日志..."
            tail -f "$ns_dir/logs/node-$node_id.log"
        fi
    fi
}

start_single_server() {
    local namespace=$1
    local node_id=$2
    local pids_file=$3

    local ns_dir=$(get_namespace_dir "$namespace")
    local config="$ns_dir/configs/servers/node-$node_id.yaml"
    local log_file="$ns_dir/logs/node-$node_id.log"

    if [[ ! -f "$config" ]]; then
        log_error "服务器节点 $node_id 的配置文件不存在: $config"
        return 1
    fi

    # 检查节点是否已在运行
    if [[ -f "$pids_file" ]]; then
        local existing_pid=$(grep "^$node_id:" "$pids_file" | cut -d: -f2)
        if [[ -n "$existing_pid" ]] && kill -0 "$existing_pid" 2>/dev/null; then
            log_warn "服务器节点 $node_id 已在运行 (PID: $existing_pid)"
            return 0
        fi
    fi

    # 启动节点
    nohup ./bin/server -c "$config" > "$log_file" 2>&1 &
    local pid=$!
    
    # 保存 PID
    echo "$node_id:$pid" >> "$pids_file"
    
    log_info "服务器节点 $node_id 已启动 (PID: $pid, 日志: $log_file)"
}

start_executors() {
    local namespace=$1
    local executor_id=${2:-"all"}
    local follow_log=${3:-false}

    check_namespace_exists "$namespace"
    
    local ns_dir=$(get_namespace_dir "$namespace")
    local pids_file="$ns_dir/.executor_pids"

    # 确保 bin/executor 存在
    if [[ ! -f "bin/executor" ]]; then
        log_warn "executor 二进制文件不存在,尝试构建..."
        make build-executor
    fi

    if [[ "$executor_id" == "all" ]]; then
        log_step "启动命名空间 '$namespace' 的所有执行器..."
        
        # 清空 PID 文件
        > "$pids_file"
        
        # 读取执行器数量配置
        local executor_count_file="$ns_dir/.executor_count"
        local num_executors=0
        if [[ -f "$executor_count_file" ]]; then
            num_executors=$(cat "$executor_count_file")
        else
            # 如果没有配置文件，默认启动所有节点对应的执行器
            num_executors=$(ls -1 "$ns_dir/configs/servers"/node-*.yaml 2>/dev/null | wc -l)
        fi
        
        # 只启动指定数量的执行器 (使用对应的节点配置)
        for ((i=0; i<$num_executors; i++)); do
            if [[ -f "$ns_dir/configs/servers/node-$i.yaml" ]]; then
                start_single_executor "$namespace" "$i" "$pids_file"
            fi
        done
        
        log_info "所有执行器已启动"
        
        if [[ "$follow_log" == true ]]; then
            log_info "查看日志: tail -f $ns_dir/logs/executor-*.log"
        fi
    else
        log_step "启动命名空间 '$namespace' 的执行器 $executor_id..."
        start_single_executor "$namespace" "$executor_id" "$pids_file"
        
        if [[ "$follow_log" == true ]]; then
            log_info "跟踪日志..."
            tail -f "$ns_dir/logs/executor-$executor_id.log"
        fi
    fi
}

start_single_executor() {
    local namespace=$1
    local executor_id=$2
    local pids_file=$3

    local ns_dir=$(get_namespace_dir "$namespace")
    # 执行器使用对应节点的配置文件
    local config="$ns_dir/configs/servers/node-$executor_id.yaml"
    local log_file="$ns_dir/logs/executor-$executor_id.log"

    if [[ ! -f "$config" ]]; then
        log_error "执行器 $executor_id 的配置文件不存在: $config"
        return 1
    fi

    # 检查执行器是否已在运行
    if [[ -f "$pids_file" ]]; then
        local existing_pid=$(grep "^$executor_id:" "$pids_file" | cut -d: -f2)
        if [[ -n "$existing_pid" ]] && kill -0 "$existing_pid" 2>/dev/null; then
            log_warn "执行器 $executor_id 已在运行 (PID: $existing_pid)"
            return 0
        fi
    fi

    # 启动执行器
    nohup ./bin/executor -c "$config" > "$log_file" 2>&1 &
    local pid=$!
    
    # 保存 PID
    echo "$executor_id:$pid" >> "$pids_file"
    
    log_info "执行器 $executor_id 已启动 (PID: $pid, 日志: $log_file)"
}

stop_servers() {
    local namespace=$1
    local node_id=${2:-"all"}

    check_namespace_exists "$namespace"
    
    local ns_dir=$(get_namespace_dir "$namespace")
    local pids_file="$ns_dir/.server_pids"

    if [[ ! -f "$pids_file" ]]; then
        log_warn "没有找到运行中的服务器节点"
        return 0
    fi

    if [[ "$node_id" == "all" ]]; then
        log_step "停止命名空间 '$namespace' 的所有服务器节点..."
        
        while IFS=: read -r nid pid; do
            if [[ -n "$pid" ]] && kill -0 "$pid" 2>/dev/null; then
                kill "$pid"
                log_info "服务器节点 $nid 已停止 (PID: $pid)"
            fi
        done < "$pids_file"
        
        rm -f "$pids_file"
        log_info "所有服务器节点已停止"
    else
        log_step "停止命名空间 '$namespace' 的服务器节点 $node_id..."
        
        local pid=$(grep "^$node_id:" "$pids_file" | cut -d: -f2)
        if [[ -z "$pid" ]]; then
            log_warn "服务器节点 $node_id 未在运行"
            return 0
        fi
        
        if kill -0 "$pid" 2>/dev/null; then
            kill "$pid"
            log_info "服务器节点 $node_id 已停止 (PID: $pid)"
        else
            log_warn "服务器节点 $node_id 的进程不存在 (PID: $pid)"
        fi
        
        # 从 PID 文件中删除该行
        grep -v "^$node_id:" "$pids_file" > "${pids_file}.tmp" || true
        mv "${pids_file}.tmp" "$pids_file"
    fi
}

stop_executors() {
    local namespace=$1
    local executor_id=${2:-"all"}

    check_namespace_exists "$namespace"
    
    local ns_dir=$(get_namespace_dir "$namespace")
    local pids_file="$ns_dir/.executor_pids"

    if [[ ! -f "$pids_file" ]]; then
        log_warn "没有找到运行中的执行器"
        return 0
    fi

    if [[ "$executor_id" == "all" ]]; then
        log_step "停止命名空间 '$namespace' 的所有执行器..."
        
        while IFS=: read -r eid pid; do
            if [[ -n "$pid" ]] && kill -0 "$pid" 2>/dev/null; then
                kill "$pid"
                log_info "执行器 $eid 已停止 (PID: $pid)"
            fi
        done < "$pids_file"
        
        rm -f "$pids_file"
        log_info "所有执行器已停止"
    else
        log_step "停止命名空间 '$namespace' 的执行器 $executor_id..."
        
        local pid=$(grep "^$executor_id:" "$pids_file" | cut -d: -f2)
        if [[ -z "$pid" ]]; then
            log_warn "执行器 $executor_id 未在运行"
            return 0
        fi
        
        if kill -0 "$pid" 2>/dev/null; then
            kill "$pid"
            log_info "执行器 $executor_id 已停止 (PID: $pid)"
        else
            log_warn "执行器 $executor_id 的进程不存在 (PID: $pid)"
        fi
        
        # 从 PID 文件中删除该行
        grep -v "^$executor_id:" "$pids_file" > "${pids_file}.tmp" || true
        mv "${pids_file}.tmp" "$pids_file"
    fi
}

# ==================== 清理功能 ====================
clean_namespace() {
    local namespace=$1

    if [[ -z "$namespace" ]]; then
        log_error "命名空间名称不能为空"
        exit 1
    fi

    local ns_dir=$(get_namespace_dir "$namespace")

    if [[ ! -d "$ns_dir" ]]; then
        log_warn "命名空间 '$namespace' 不存在"
        return 0
    fi

    # 先停止所有服务器和执行器
    if [[ -f "$ns_dir/.server_pids" ]]; then
        log_step "停止运行中的服务器节点..."
        stop_servers "$namespace" "all"
    fi
    
    if [[ -f "$ns_dir/.executor_pids" ]]; then
        log_step "停止运行中的执行器..."
        stop_executors "$namespace" "all"
    fi

    # 清理数据
    log_step "清理命名空间 '$namespace'..."
    rm -rf "$ns_dir"
    log_info "命名空间 '$namespace' 已清理"
}

# ==================== 列表功能 ====================
list_namespaces() {
    log_step "已创建的命名空间:"
    
    if [[ ! -d "$WORKSPACE_DIR" ]]; then
        log_info "没有找到任何命名空间"
        return 0
    fi

    local count=0
    for ns in "$WORKSPACE_DIR"/*; do
        if [[ -d "$ns" ]]; then
            local name=$(basename "$ns")
            local num_nodes=$(ls -1 "$ns/configs/servers" 2>/dev/null | wc -l)
            
            # 检查是否有运行中的服务器和执行器
            local running=""
            local server_count=0
            local executor_count=0
            
            if [[ -f "$ns/.server_pids" ]]; then
                while IFS=: read -r nid pid; do
                    if [[ -n "$pid" ]] && kill -0 "$pid" 2>/dev/null; then
                        ((server_count++))
                    fi
                done < "$ns/.server_pids"
            fi
            
            if [[ -f "$ns/.executor_pids" ]]; then
                while IFS=: read -r eid pid; do
                    if [[ -n "$pid" ]] && kill -0 "$pid" 2>/dev/null; then
                        ((executor_count++))
                    fi
                done < "$ns/.executor_pids"
            fi
            
            if [[ $server_count -gt 0 ]] || [[ $executor_count -gt 0 ]]; then
                running=" ${GREEN}(运行中: ${server_count}服务器, ${executor_count}执行器)${NC}"
            fi
            
            echo -e "  - ${BLUE}$name${NC}: $num_nodes 节点$running"
            ((count++))
        fi
    done

    if [[ $count -eq 0 ]]; then
        log_info "没有找到任何命名空间"
    fi
}

# ==================== 状态查看功能 ====================
status_namespace() {
    local namespace=$1

    check_namespace_exists "$namespace"
    
    local ns_dir=$(get_namespace_dir "$namespace")
    local pids_file="$ns_dir/.node_pids"

    log_step "命名空间 '$namespace' 状态:"
    echo "  目录: $ns_dir"
    echo "  服务器节点数: $(ls -1 "$ns_dir/configs/servers" 2>/dev/null | wc -l)"
    echo ""
    
    # 显示运行中的服务器节点
    local server_pids="$ns_dir/.server_pids"
    if [[ -f "$server_pids" ]]; then
        log_step "运行中的服务器节点:"
        local has_running=false
        while IFS=: read -r nid pid; do
            if [[ -n "$pid" ]] && kill -0 "$pid" 2>/dev/null; then
                echo -e "  - 服务器节点 ${GREEN}$nid${NC} (PID: $pid)"
                has_running=true
            fi
        done < "$server_pids"
        
        if [[ "$has_running" == false ]]; then
            echo "  (无运行中的服务器节点)"
        fi
    else
        echo "  (无运行中的服务器节点)"
    fi
    
    echo ""
    
    # 显示运行中的执行器
    local executor_pids="$ns_dir/.executor_pids"
    if [[ -f "$executor_pids" ]]; then
        log_step "运行中的执行器:"
        local has_running=false
        while IFS=: read -r eid pid; do
            if [[ -n "$pid" ]] && kill -0 "$pid" 2>/dev/null; then
                echo -e "  - 执行器 ${GREEN}$eid${NC} (PID: $pid)"
                has_running=true
            fi
        done < "$executor_pids"
        
        if [[ "$has_running" == false ]]; then
            echo "  (无运行中的执行器)"
        fi
    else
        echo "  (无运行中的执行器)"
    fi
}

# ==================== 日志查看功能 ====================
logs_namespace() {
    local namespace=$1
    local node_id=${2:-"all"}
    local follow=${3:-false}

    check_namespace_exists "$namespace"
    
    local ns_dir=$(get_namespace_dir "$namespace")

    if [[ "$node_id" == "all" ]]; then
        if [[ "$follow" == true ]]; then
            tail -f "$ns_dir/logs"/node-*.log
        else
            tail -n 50 "$ns_dir/logs"/node-*.log
        fi
    else
        local log_file="$ns_dir/logs/node-$node_id.log"
        if [[ ! -f "$log_file" ]]; then
            log_error "节点 $node_id 的日志文件不存在: $log_file"
            exit 1
        fi
        
        if [[ "$follow" == true ]]; then
            tail -f "$log_file"
        else
            tail -n 50 "$log_file"
        fi
    fi
}

# ==================== 帮助信息 ====================
show_help() {
    cat << EOF
POT 区块链节点管理脚本

用法: $0 <命令> [参数...]

命令:
  init <namespace> <num_nodes> <num_executors>
      初始化命名空间,创建配置文件和密钥
      执行器数量不能超过节点数量
      示例: $0 init mytest 4 2

  start <namespace> [server|executor|all] [id] [-f|--follow]
      启动服务器节点或执行器
      - 不指定类型: 启动所有服务器和执行器
      - server [id]: 启动服务器节点（不指定id则启动所有）
      - executor [id]: 启动执行器（不指定id则启动所有）
      - all: 启动所有服务器和执行器
      - -f/--follow: 跟踪日志输出
      示例: $0 start mytest              # 启动所有
      示例: $0 start mytest server      # 启动所有服务器
      示例: $0 start mytest server 0    # 启动服务器0
      示例: $0 start mytest executor    # 启动所有执行器
      示例: $0 start mytest executor 0 -f # 启动执行器0并跟踪日志

  stop <namespace> [server|executor|all] [id]
      停止服务器节点或执行器
      - 不指定类型: 停止所有服务器和执行器
      - server [id]: 停止服务器节点（不指定id则停止所有）
      - executor [id]: 停止执行器（不指定id则停止所有）
      - all: 停止所有服务器和执行器
      示例: $0 stop mytest              # 停止所有
      示例: $0 stop mytest server      # 停止所有服务器
      示例: $0 stop mytest server 0    # 停止服务器0
      示例: $0 stop mytest executor    # 停止所有执行器

  status <namespace>
      查看命名空间状态
      示例: $0 status mytest

  logs <namespace> [node_id] [-f|--follow]
      查看日志
      - 不指定 node_id: 查看所有节点日志
      - 指定 node_id: 查看指定节点日志
      - -f/--follow: 实时跟踪日志
      示例: $0 logs mytest
      示例: $0 logs mytest 0 -f

  clean <namespace>
      清理命名空间(包括停止节点、删除所有数据)
      示例: $0 clean mytest

  list
      列出所有命名空间
      示例: $0 list

  test
      快速创建测试命名空间(1节点1执行器)
      示例: $0 test

  help
      显示此帮助信息

示例工作流:
  1. 初始化测试环境:
     $0 init dev 4 2

  2. 启动所有节点:
     $0 start dev

  3. 查看状态:
     $0 status dev

  4. 查看日志:
     $0 logs dev 0 -f

  5. 停止所有节点:
     $0 stop dev

  6. 清理环境:
     $0 clean dev

EOF
}

# ==================== 主函数 ====================
main() {
    # 检查基本依赖
    check_templates

    # 解析命令
    local command=${1:-"help"}
    shift || true

    case "$command" in
        init)
            if [[ $# -lt 3 ]]; then
                log_error "用法: $0 init <namespace> <num_nodes> <num_executors>"
                exit 1
            fi
            init_namespace "$1" "$2" "$3"
            ;;
        
        start)
            if [[ $# -lt 1 ]]; then
                log_error "用法: $0 start <namespace> [server|executor|all] [id] [-f|--follow]"
                exit 1
            fi
            local namespace=$1
            local type="all"
            local id="all"
            local follow=false
            shift
            
            # 解析参数
            while [[ $# -gt 0 ]]; do
                case "$1" in
                    -f|--follow)
                        follow=true
                        shift
                        ;;
                    server|executor|all)
                        type=$1
                        shift
                        # 检查是否有ID参数
                        if [[ $# -gt 0 ]] && [[ ! "$1" =~ ^- ]]; then
                            id=$1
                            shift
                        fi
                        ;;
                    *)
                        # 如果还没设置type，这可能是旧的node_id参数
                        if [[ "$type" == "all" ]] && [[ ! "$1" =~ ^- ]]; then
                            id=$1
                            type="server"
                        fi
                        shift
                        ;;
                esac
            done
            
            # 根据类型启动
            case "$type" in
                server)
                    start_servers "$namespace" "$id" "$follow"
                    ;;
                executor)
                    start_executors "$namespace" "$id" "$follow"
                    ;;
                all)
                    start_servers "$namespace" "all" false
                    start_executors "$namespace" "all" false
                    log_info "使用 './run.sh stop $namespace' 停止所有服务"
                    if [[ "$follow" == true ]]; then
                        log_info "跟踪日志: tail -f $(get_namespace_dir "$namespace")/logs/*.log"
                    fi
                    ;;
            esac
            ;;
        
        stop)
            if [[ $# -lt 1 ]]; then
                log_error "用法: $0 stop <namespace> [server|executor|all] [id]"
                exit 1
            fi
            local namespace=$1
            local type="all"
            local id="all"
            shift
            
            # 解析参数
            if [[ $# -gt 0 ]]; then
                case "$1" in
                    server|executor|all)
                        type=$1
                        shift
                        if [[ $# -gt 0 ]]; then
                            id=$1
                        fi
                        ;;
                    *)
                        # 如果不是关键字，当作server id处理（向后兼容）
                        id=$1
                        type="server"
                        ;;
                esac
            fi
            
            # 根据类型停止
            case "$type" in
                server)
                    stop_servers "$namespace" "$id"
                    ;;
                executor)
                    stop_executors "$namespace" "$id"
                    ;;
                all)
                    stop_servers "$namespace" "all"
                    stop_executors "$namespace" "all"
                    ;;
            esac
            ;;
        
        status)
            if [[ $# -lt 1 ]]; then
                log_error "用法: $0 status <namespace>"
                exit 1
            fi
            status_namespace "$1"
            ;;
        
        logs)
            if [[ $# -lt 1 ]]; then
                log_error "用法: $0 logs <namespace> [node_id] [-f|--follow]"
                exit 1
            fi
            local namespace=$1
            local node_id="all"
            local follow=false
            shift
            
            while [[ $# -gt 0 ]]; do
                case "$1" in
                    -f|--follow)
                        follow=true
                        shift
                        ;;
                    *)
                        node_id=$1
                        shift
                        ;;
                esac
            done
            
            logs_namespace "$namespace" "$node_id" "$follow"
            ;;
        
        clean)
            if [[ $# -lt 1 ]]; then
                log_error "用法: $0 clean <namespace>"
                exit 1
            fi
            clean_namespace "$1"
            ;;
        
        list)
            list_namespaces
            ;;
        
        test)
            log_step "创建默认测试命名空间 (1节点, 1执行器)..."
            init_namespace "$DEFAULT_NAMESPACE" 1 1
            log_info "使用 '$0 start $DEFAULT_NAMESPACE' 启动测试节点"
            ;;
        
        help|--help|-h)
            show_help
            ;;
        
        *)
            log_error "未知命令: $command"
            echo ""
            show_help
            exit 1
            ;;
    esac
}

# 运行主函数
main "$@"
