# 共识可视化大屏布局设计方案

## 一、整体布局结构

### 布局方案：中心网络拓扑 + 周边监控面板

```
┌─────────────────────────────────────────────────────────────────┐
│                    【顶部状态栏】                                  │
│  系统运行时长 | 总区块高度 | 当前TPS | 节点状态 | 网络状态           │
└─────────────────────────────────────────────────────────────────┘
┌────────────┬─────────────────────────────────────┬────────────┐
│            │                                     │            │
│  【左侧】   │        【中心网络拓扑图】             │  【右侧】   │
│            │                                     │            │
│ POT共识    │     ┌─────────────────┐            │  交易池    │
│ 状态       │     │  委员会层(上层)   │            │  监控      │
│            │     │  ○──○──○──○     │            │            │
│ VDF计算    │     │   (节点数少)     │            │  BCI激励   │
│ 监控       │     └─────────────────┘            │            │
│            │                                     │            │
│ 性能指标   │     ┌─────────────────┐            │  存储状态  │
│            │     │  POT节点层(下层) │            │            │
├────────────┤     │  ●══●══●══●     │            ├────────────┤
│            │     │  ‖  ‖  ‖  ‖     │            │            │
│  【底部】   │     │  ●══●══●══●     │            │  【底部】   │
│  委员会    │     │   (节点数多)     │            │  系统      │
│  详情      │     └─────────────────┘            │  日志      │
│            │                                     │            │
└────────────┴─────────────────────────────────────┴────────────┘
```

### 布局说明：
- **中心区域（60%宽度）**: 网络拓扑图，双层结构
  - 上层：委员会节点（4-8个节点，Leader高亮）
  - 下层：POT节点集群（10-20个节点，密集连接）
  - 层间连接：虚线表示委员会与POT节点的关系
  
- **左侧面板（20%宽度）**: 上中下三个区块
  - 上：POT共识状态（Epoch、难度、挖矿状态）
  - 中：VDF计算监控（进度条、性能指标）
  - 下：性能指标（TPS、延迟、队列）
  
- **右侧面板（20%宽度）**: 上中下三个区块
  - 上：交易池监控（交易量、类型分布）
  - 中：BCI激励（奖励分配、利率）
  - 下：存储状态（容量、趋势）
  
- **底部区域（左右分割）**:
  - 左：委员会详细信息（成员列表、分片状态）
  - 右：系统日志/事件流

## 二、各板块详细设计

### 【顶部状态栏】 - 全局关键指标
**类型**: 横向指标卡片组
**指标**:
- 系统运行时长（天时分秒）
- 当前区块高度（大数字显示）
- 实时TPS（动态数字，带趋势箭头）
- 在线节点数/总节点数（带状态灯）
- 网络健康度（百分比圆环）

**可视化组件**: 
- 大数字卡片 + 趋势指示器
- 状态灯（绿/黄/红）

---

### 【中心：网络拓扑图】
**位置**: 页面中心（60%宽度）
**类型**: 双层网络拓扑图

**主要可视化元素**:

#### 上层：委员会节点层
1. **节点展示**:
   - Leader节点：大圆形，金色边框，带皇冠图标
   - 成员节点：中等圆形，绿色边框，带成员标识
   - 节点数量：4-8个（根据CommitteeLen）
   - 节点信息：悬浮显示地址、状态、工作高度

2. **节点连接**:
   - Leader与成员：粗实线，表示强连接
   - 成员之间：细实线，表示通信
   - 连接动画：数据流动效果

3. **状态指示**:
   - 工作中：绿色脉冲光晕
   - 空闲：灰色
   - 异常：红色闪烁

#### 下层：POT节点集群
1. **节点展示**:
   - 节点样式：中小圆形，蓝色渐变
   - 节点数量：10-20个（动态）
   - 挖矿节点：带闪电图标
   - 普通节点：圆点样式

2. **节点连接**:
   - P2P连接：细线网状结构
   - 连接强度：线条粗细表示通信频率
   - 延迟显示：线条颜色（绿/黄/红）

3. **集群布局**:
   - 使用力导向图算法
   - 相近节点聚合
   - 支持拖拽调整

#### 层间连接
1. **委员会-POT连接**:
   - 虚线连接，表示监管关系
   - Leader连接到多个POT节点
   - 数据同步动画

2. **交互功能**:
   - 点击节点查看详情
   - 框选多个节点
   - 缩放和平移
   - 高亮连接路径

**数据来源API**: 
- `GET /api/network/topology` - 获取所有节点和连接
- `GET /api/committee/status` - 获取委员会信息
- `GET /api/pot/status` - 获取POT节点状态
- WebSocket实时更新节点状态

**推荐组件**:
- @antv/g6 - 网络拓扑图核心库
- 自定义节点和边渲染器
- 动画效果插件

**关键配置**:
```javascript
// G6 配置示例
{
  // 上层委员会布局
  committeeLayout: {
    type: 'circular',
    radius: 150,
    center: [width/2, height*0.3]
  },
  
  // 下层POT节点布局
  potLayout: {
    type: 'force',
    center: [width/2, height*0.7],
    linkDistance: 100,
    nodeStrength: -300,
    edgeStrength: 0.5
  },
  
  // 节点样式
  nodeStyles: {
    leader: {
      size: 60,
      fill: '#ffd700',
      stroke: '#ff8800',
      lineWidth: 3
    },
    committee: {
      size: 45,
      fill: '#00ff88',
      stroke: '#00cc66',
      lineWidth: 2
    },
    potMining: {
      size: 35,
      fill: '#00d4ff',
      stroke: '#0099cc',
      lineWidth: 2
    },
    potNormal: {
      size: 25,
      fill: '#4488ff',
      stroke: '#3366cc',
      lineWidth: 1
    }
  }
}
```

---

### 【左上：POT共识状态】
**位置**: 左侧上部
**类型**: 紧凑型仪表盘

### 【左上：POT共识状态】
**位置**: 左侧上部
**类型**: 紧凑型仪表盘

**主要可视化元素**:
1. **Epoch进度环** - 小型圆环进度条
2. **挖矿难度** - 数字卡片 + 趋势箭头
3. **挖矿状态** - 状态指示灯（工作中/空闲）
4. **关键数据**:
   - 当前区块高度
   - Nonce值
   - 叔块数量

**数据来源API**: 
- `GET /api/pot/status`

**推荐组件**:
- Ant Design Vue Progress（进度环）
- Statistic（数字统计卡片）
- Badge（状态徽章）

---

### 【左中：VDF计算监控】
**位置**: 左侧中部
**类型**: 进度监控面板

**主要可视化元素**:
1. **VDF0进度** - 大型圆环（百分比 + 迭代次数）
2. **VDF1并行线程** - 4个小型进度条
3. **VDFhalf进度** - 横向进度条
4. **性能指标**:
   - 平均计算耗时
   - 验证失败次数

**数据来源API**: 
- `GET /api/pot/vdf`

**推荐组件**:
- Ant Design Vue Progress
- 自定义进度圆环

---

### 【左下：性能指标】
**位置**: 左侧下部
**类型**: 实时监控卡片

**主要可视化元素**:
1. **关键指标**:
   - 当前TPS（大数字）
   - 平均出块时间
   - 区块确认延迟
2. **消息队列** - 进度条显示队列长度
3. **健康度雷达图** - 小型雷达图

**数据来源API**: 
- `GET /api/system/overview`
- `GET /api/metrics/history`

**推荐组件**:
- Ant Design Vue Statistic
- ECharts 迷你雷达图

---

### 【右上：交易池监控】
**位置**: 右侧上部
**类型**: 实时数据流图表

**主要可视化元素**:
1. **交易池饼图** - 小型饼图（Marked/Unmarked）
2. **交易类型柱状图** - 横向迷你柱状图
3. **关键指标**:
   - 交易池总大小
   - 待处理交易数
   - 平均确认时间

**数据来源API**: 
- `GET /api/mempool/status`

**推荐组件**:
- ECharts 饼图/柱状图
- Ant Design Vue Statistic

---

### 【右中：BCI激励】
**位置**: 右侧中部
**类型**: 财务指标面板

**主要可视化元素**:
1. **奖励分配** - 小型饼图或堆叠柱状图
2. **关键数据**:
   - 总奖励金额
   - 锁定激励金额
   - 待分配奖励

**数据来源API**: 
- `GET /api/bci/status`

**推荐组件**:
- ECharts 饼图
- Ant Design Vue Statistic

---

### 【右下：存储状态】
**位置**: 右侧下部
**类型**: 存储监控面板

**主要可视化元素**:
1. **容量仪表盘** - 小型仪表盘组
2. **关键指标**:
   - 总存储容量
   - 使用率
   - 区块数量

**数据来源API**: 
- `GET /api/storage/status`

**推荐组件**:
- ECharts 仪表盘
- Ant Design Vue Progress

---

### 【底部左：委员会详情】
**位置**: 底部左侧
**类型**: 信息列表和表格

**主要可视化元素**:
1. **委员会信息**:
   - 共识类型标签（Simple/CR/Whirly）
   - 委员会大小、数量
   - 批处理大小
   - 确认延迟、超时配置
2. **工作阶段** - 步骤条显示当前阶段：
   初始化 → 置换 → 抽选 → 份额分发 → 共识工作
3. **分片状态表格**:
   - 分片名称/ID
   - Leader地址
   - 工作高度
   - 消息队列长度
   - 选举区块高度
   - 自身身份
   - 状态

**数据来源API**: 
- `GET /api/committee/status`

**推荐组件**:
- Ant Design Vue Descriptions（描述列表）
- Steps（步骤条）
- Table（数据表格）

---

### 【底部右：系统日志/事件流】
**位置**: 底部右侧
**类型**: 实时日志流

**主要可视化元素**:
1. **事件时间线** - 重要事件标记
   - 新区块产生
   - 委员会切换
   - VDF完成
   - 异常告警
2. **日志列表** - 虚拟滚动列表
   - 时间戳
   - 日志级别（Info/Warning/Error）
   - 日志内容
3. **筛选器**:
   - 按类型筛选（POT/Committee/BCI/Network）
   - 按级别筛选
   - 搜索功能

**数据来源**: 
- WebSocket实时推送
- `GET /api/system/events`

**推荐组件**:
- Ant Design Vue Timeline
- List with virtual scroll
- Tag（日志级别标签）

---

## 三、技术栈建议

### 前端框架
- **Vue 3.3.4** + TypeScript 4.5.4
- **Ant Design Vue 4.1.2** - UI组件库
- **@ant-design/icons-vue 6.1.0** - 图标库
- **Vite 2.9.16** - 构建工具
- **Pinia 2.0.23** - 状态管理
- **WebSocket** - 实时数据推送

### 可视化库
- **@antv/g6 4.x** - 网络拓扑图（核心）
- **ECharts 5.x** - 基础图表（仪表盘、折线图、饼图等）
- **vue-echarts** - Vue ECharts 集成
- **CountUp.js** - 数字动画

### 工具库
- **axios** - HTTP 客户端
- **@vueuse/core** - Vue 组合式工具
- **dayjs** - 时间处理

### 样式
- **TailwindCSS 3.x** - 快速布局
- **暗色主题** - 大屏常用
- **响应式设计** - 适配不同屏幕

---

## 四、颜色方案建议

### 主题色
- **背景色**: `#0a0e27` (深蓝黑)
- **卡片背景**: `#141d3a` (深蓝)
- **边框色**: `#1e3a5f` (蓝灰)

### 功能色
- **委员会Leader**: `#ffd700` (金色)
- **委员会成员**: `#00ff88` (翠绿)
- **POT挖矿节点**: `#00d4ff` (青蓝)
- **POT普通节点**: `#4488ff` (蓝色)
- **交易**: `#ffa500` (橙色)
- **激励**: `#ffd700` (金色)
- **存储**: `#9370db` (紫色)
- **VDF**: `#ff69b4` (粉红)

### 状态色
- **正常/在线**: `#00ff00` (绿)
- **警告**: `#ffff00` (黄)
- **错误/离线**: `#ff0000` (红)
- **空闲**: `#808080` (灰)

### 连接线颜色
- **强连接**: `#00ff88` (绿)
- **中等连接**: `#ffa500` (橙)
- **弱连接/高延迟**: `#ff4444` (红)
- **层间虚线**: `#888888` (灰)

---

## 五、交互设计

### 网络拓扑交互
- **节点悬浮**: 显示节点详细信息（地址、状态、连接数）
- **节点点击**: 高亮该节点及其所有连接
- **边悬浮**: 显示延迟、带宽等信息
- **缩放**: 鼠标滚轮缩放
- **平移**: 拖拽画布移动
- **框选**: 按住Shift框选多个节点
- **聚焦**: 双击节点居中聚焦

### 侧边面板交互
- **卡片悬浮**: 显示更多详细信息
- **指标点击**: 展开历史趋势图
- **进度条**: 实时动画更新

### 实时更新策略
- **网络拓扑**: WebSocket实时更新节点状态（1秒）
- **POT状态**: 5秒刷新
- **VDF进度**: 1秒刷新
- **交易池**: 5秒刷新
- **其他面板**: 10-30秒刷新

### 报警提示
- **异常节点**: 红色闪烁
- **连接中断**: 虚线显示
- **队列过载**: 进度条变红 + 警告图标
- **全局通知**: 顶部通知栏（可配置声音提醒）

---

## 六、响应式布局

### 大屏 (>1920px)
- 中心拓扑图 60%
- 左右侧边栏各 20%
- 底部面板 20% 高度

### 中屏 (1280-1920px)
- 中心拓扑图 50%
- 左右侧边栏各 25%
- 底部面板可折叠

### 小屏 (<1280px)
- 顶部：状态栏
- 中间：网络拓扑图（全宽）
- 底部：标签页切换各个面板

### 移动端 (不推荐，仅查看)
- 单列布局
- 网络拓扑简化版
- 关键指标优先显示

---

## 七、G6 网络拓扑核心实现

### 双层布局配置
```javascript
import G6 from '@antv/g6';

// 自定义双层布局算法
G6.registerLayout('dual-layer', {
  execute() {
    const nodes = this.nodes;
    const committeeNodes = nodes.filter(n => n.type === 'committee');
    const potNodes = nodes.filter(n => n.type === 'pot');
    
    // 上层：委员会圆形布局
    const committeeRadius = 200;
    const center = [width / 2, height * 0.25];
    committeeNodes.forEach((node, i) => {
      const angle = (2 * Math.PI * i) / committeeNodes.length;
      node.x = center[0] + committeeRadius * Math.cos(angle);
      node.y = center[1] + committeeRadius * Math.sin(angle);
    });
    
    // 下层：POT力导向布局
    // 使用内置的force算法
    const forceLayout = new G6.Layout.force({
      center: [width / 2, height * 0.65],
      linkDistance: 80,
      nodeStrength: -200,
      edgeStrength: 0.3,
      preventOverlap: true,
      nodeSize: 30
    });
    forceLayout.init({ nodes: potNodes });
    forceLayout.execute();
  }
});

// 图实例创建
const graph = new G6.Graph({
  container: 'topology-container',
  width,
  height,
  layout: {
    type: 'dual-layer'
  },
  modes: {
    default: ['drag-canvas', 'zoom-canvas', 'drag-node', 'click-select']
  },
  defaultNode: {
    style: {
      fill: '#4488ff',
      stroke: '#3366cc',
      lineWidth: 2
    },
    labelCfg: {
      position: 'bottom',
      style: {
        fill: '#fff',
        fontSize: 12
      }
    }
  },
  defaultEdge: {
    style: {
      stroke: '#888',
      lineWidth: 1,
      opacity: 0.6
    }
  },
  nodeStateStyles: {
    active: {
      fill: '#00ff88',
      stroke: '#00cc66',
      lineWidth: 3,
      shadowColor: '#00ff88',
      shadowBlur: 20
    },
    inactive: {
      opacity: 0.3
    }
  },
  edgeStateStyles: {
    active: {
      stroke: '#00d4ff',
      lineWidth: 2,
      opacity: 1
    }
  }
});
```

### 自定义节点渲染
```javascript
// 注册委员会Leader节点
G6.registerNode('leader-node', {
  draw(cfg, group) {
    // 外圈光晕
    group.addShape('circle', {
      attrs: {
        r: 35,
        fill: 'l(0) 0:#ffd700 1:#ff8800',
        opacity: 0.3
      }
    });
    
    // 主圆形
    const circle = group.addShape('circle', {
      attrs: {
        r: 25,
        fill: '#ffd700',
        stroke: '#ff8800',
        lineWidth: 3,
        shadowColor: '#ffd700',
        shadowBlur: 15
      },
      name: 'circle-shape'
    });
    
    // 皇冠图标
    group.addShape('text', {
      attrs: {
        text: '👑',
        fontSize: 20,
        x: 0,
        y: 5,
        textAlign: 'center',
        textBaseline: 'middle'
      }
    });
    
    // 标签
    group.addShape('text', {
      attrs: {
        text: cfg.label,
        x: 0,
        y: 40,
        fontSize: 12,
        fill: '#fff',
        textAlign: 'center'
      }
    });
    
    return circle;
  },
  
  // 更新节点
  update(cfg, node) {
    const group = node.getContainer();
    const shape = group.get('children')[1]; // 主圆形
    
    // 根据状态更新样式
    if (cfg.status === 'active') {
      shape.attr('fill', '#00ff00');
    } else if (cfg.status === 'error') {
      shape.attr('fill', '#ff0000');
    }
  }
});

// 注册POT挖矿节点
G6.registerNode('pot-mining-node', {
  draw(cfg, group) {
    // 主圆形
    const circle = group.addShape('circle', {
      attrs: {
        r: 18,
        fill: '#00d4ff',
        stroke: '#0099cc',
        lineWidth: 2
      },
      name: 'circle-shape'
    });
    
    // 闪电图标
    group.addShape('text', {
      attrs: {
        text: '⚡',
        fontSize: 14,
        x: 0,
        y: 3,
        textAlign: 'center',
        textBaseline: 'middle'
      }
    });
    
    return circle;
  }
});
```

### 动画效果
```javascript
// 数据流动动画
function animateEdge(edge) {
  const shape = edge.getKeyShape();
  let index = 0;
  
  shape.animate(
    () => {
      index++;
      if (index > 100) index = 0;
      
      return {
        lineDash: [4, 2, 1, 2],
        lineDashOffset: -index
      };
    },
    {
      repeat: true,
      duration: 3000
    }
  );
}

// 节点脉冲动画
function pulseNode(node) {
  const group = node.getContainer();
  const shape = group.get('children')[0]; // 光晕
  
  shape.animate(
    {
      r: 40,
      opacity: 0
    },
    {
      duration: 2000,
      repeat: true,
      easing: 'easeCubic'
    }
  );
}
```

---

## 八、Ant Design Vue 组件使用示例

### 状态统计卡片
```vue
<template>
  <a-card :bordered="false" class="stat-card">
    <a-statistic
      title="当前TPS"
      :value="currentTPS"
      :precision="2"
      :value-style="{ color: '#00d4ff' }"
    >
      <template #prefix>
        <ThunderboltOutlined />
      </template>
      <template #suffix>
        <a-tag :color="tpsStatus">
          {{ tpsTrend }}
        </a-tag>
      </template>
    </a-statistic>
  </a-card>
</template>

<script setup lang="ts">
import { ThunderboltOutlined } from '@ant-design/icons-vue';
import { computed } from 'vue';

const props = defineProps<{
  currentTPS: number;
}>();

const tpsStatus = computed(() => {
  if (props.currentTPS > 100) return 'success';
  if (props.currentTPS > 50) return 'warning';
  return 'error';
});

const tpsTrend = computed(() => {
  // 计算趋势
  return '↑ 15%';
});
</script>
```

### 进度环组件
```vue
<template>
  <div class="progress-wrapper">
    <a-progress
      type="circle"
      :percent="progress"
      :stroke-color="strokeColor"
      :width="120"
    >
      <template #format="percent">
        <span class="progress-text">{{ percent }}%</span>
        <div class="progress-label">{{ label }}</div>
      </template>
    </a-progress>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';

const props = defineProps<{
  progress: number;
  label: string;
  color?: string;
}>();

const strokeColor = computed(() => ({
  '0%': props.color || '#00d4ff',
  '100%': props.color || '#0099cc'
}));
</script>
```

### 委员会信息描述
```vue
<template>
  <a-descriptions
    title="委员会信息"
    :column="2"
    bordered
    size="small"
  >
    <a-descriptions-item label="共识类型">
      <a-tag color="blue">{{ consensusType }}</a-tag>
    </a-descriptions-item>
    <a-descriptions-item label="委员会大小">
      {{ committeeSize }}
    </a-descriptions-item>
    <a-descriptions-item label="批处理大小">
      {{ batchSize }}
    </a-descriptions-item>
    <a-descriptions-item label="确认延迟">
      {{ confirmDelay }} 块
    </a-descriptions-item>
  </a-descriptions>
</template>
```

### 工作阶段步骤条
```vue
<template>
  <a-steps :current="currentStep" size="small">
    <a-step title="初始化" />
    <a-step title="置换" />
    <a-step title="抽选" />
    <a-step title="份额分发" />
    <a-step title="共识工作" />
  </a-steps>
</template>

<script setup lang="ts">
import { ref } from 'vue';

const currentStep = ref(2); // 当前在"抽选"阶段
</script>
```

### 分片状态表格
```vue
<template>
  <a-table
    :columns="columns"
    :data-source="shardingData"
    :pagination="false"
    size="small"
    :scroll="{ y: 300 }"
  >
    <template #bodyCell="{ column, record }">
      <template v-if="column.key === 'status'">
        <a-badge
          :status="getStatusType(record.status)"
          :text="record.status"
        />
      </template>
      <template v-if="column.key === 'identity'">
        <a-tag :color="getIdentityColor(record.identity)">
          {{ record.identity }}
        </a-tag>
      </template>
    </template>
  </a-table>
</template>

<script setup lang="ts">
const columns = [
  { title: '分片ID', dataIndex: 'id', key: 'id' },
  { title: 'Leader', dataIndex: 'leader', key: 'leader' },
  { title: '工作高度', dataIndex: 'height', key: 'height' },
  { title: '队列长度', dataIndex: 'queueLen', key: 'queueLen' },
  { title: '身份', dataIndex: 'identity', key: 'identity' },
  { title: '状态', dataIndex: 'status', key: 'status' }
];

function getStatusType(status: string) {
  const map: Record<string, string> = {
    'active': 'processing',
    'idle': 'default',
    'error': 'error'
  };
  return map[status] || 'default';
}

function getIdentityColor(identity: string) {
  const map: Record<string, string> = {
    'Leader': 'gold',
    'Member': 'green',
    'Observer': 'blue'
  };
  return map[identity] || 'default';
}
</script>
```

---

## 九、开发注意事项

### 性能优化
1. **网络拓扑**:
   - 节点数量>50时，降低渲染频率
   - 使用节点聚合（相近节点合并显示）
   - 实现LOD（Level of Detail）机制

2. **图表渲染**:
   - ECharts使用`notMerge: false`增量更新
   - 限制历史数据点数量（最多200个点）

3. **数据更新**:
   - WebSocket批量更新，避免频繁刷新
   - 使用防抖/节流控制更新频率

### 数据准确性
1. 所有数值保留2位小数
2. 大数字使用千分位分隔
3. 时间统一使用ISO 8601格式
4. 百分比范围0-100

### 错误处理
1. API超时时间: 10秒
2. 失败重试: 最多3次
3. WebSocket断线自动重连
4. 降级方案: 显示历史数据/默认值

---

最后更新: 2025-11-08

