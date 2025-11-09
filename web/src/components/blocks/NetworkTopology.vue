<template>
  <div class="network-topology card h-full relative">
    <div class="flex items-center justify-between mb-4">
      <h3 class="text-lg font-semibold flex items-center gap-2">
        <NodeIndexOutlined />
        网络拓扑
      </h3>
      
      <!-- 工具栏 -->
      <div class="flex items-center gap-2">
        <a-tooltip title="放大">
          <a-button size="small" @click="zoomIn">
            <template #icon><ZoomInOutlined /></template>
          </a-button>
        </a-tooltip>
        <a-tooltip title="缩小">
          <a-button size="small" @click="zoomOut">
            <template #icon><ZoomOutOutlined /></template>
          </a-button>
        </a-tooltip>
        <a-tooltip title="适应画布">
          <a-button size="small" @click="fitView">
            <template #icon><FullscreenOutlined /></template>
          </a-button>
        </a-tooltip>
      </div>
    </div>
    
    <!-- G6 容器 -->
    <div ref="container" class="topology-container" style="height: calc(100% - 60px);"></div>
    
    <!-- 图例 -->
    <div class="legend absolute bottom-4 right-4 bg-bg-primary bg-opacity-80 p-3 rounded border border-border">
      <div class="text-xs space-y-2">
        <div class="flex items-center gap-2">
          <span class="w-3 h-3 rounded-full bg-committee-leader"></span>
          <span>委员会Leader</span>
        </div>
        <div class="flex items-center gap-2">
          <span class="w-3 h-3 rounded-full bg-committee-member"></span>
          <span>委员会成员</span>
        </div>
        <div class="flex items-center gap-2">
          <span class="w-3 h-3 rounded-full bg-pot-mining"></span>
          <span>POT挖矿节点</span>
        </div>
        <div class="flex items-center gap-2">
          <span class="w-3 h-3 rounded-full bg-pot-normal"></span>
          <span>POT普通节点</span>
        </div>
      </div>
    </div>
    
    <!-- 节点详情弹窗 -->
    <a-modal
      v-model:open="detailVisible"
      title="节点详情"
      :footer="null"
      width="500px"
    >
      <div v-if="selectedNode" class="space-y-3">
        <a-descriptions :column="1" size="small" bordered>
          <a-descriptions-item label="节点ID">
            {{ selectedNode.id }}
          </a-descriptions-item>
          <a-descriptions-item label="地址">
            {{ formatAddress(selectedNode.address) }}
          </a-descriptions-item>
          <a-descriptions-item label="类型">
            <a-tag :color="getNodeTypeColor(selectedNode.type)">
              {{ getNodeTypeName(selectedNode.type) }}
            </a-tag>
          </a-descriptions-item>
          <a-descriptions-item label="状态">
            <a-badge :status="getStatusType(selectedNode.status)" :text="selectedNode.status" />
          </a-descriptions-item>
          <a-descriptions-item label="连接数">
            {{ selectedNode.connections || 0 }}
          </a-descriptions-item>
          <a-descriptions-item label="延迟">
            {{ selectedNode.latency || 0 }}ms
          </a-descriptions-item>
        </a-descriptions>
      </div>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import { storeToRefs } from 'pinia'
import {
  NodeIndexOutlined,
  ZoomInOutlined,
  ZoomOutOutlined,
  FullscreenOutlined
} from '@ant-design/icons-vue'
import { useNetworkStore } from '@/stores/network'
import { formatAddress } from '@/utils/format'

const container = ref<HTMLDivElement>()
const networkStore = useNetworkStore()
const { topology } = storeToRefs(networkStore)

const detailVisible = ref(false)
const selectedNode = ref<any>(null)

let graph: any = null

function initGraph() {
  if (!container.value) return
  
  // 动态导入 G6 (安装依赖后会生效)
  import('@antv/g6').then((G6Module) => {
    const G6 = G6Module.default || G6Module
    
    // 注册委员会组节点
    G6.registerNode('committee-group', {
      draw(cfg: any, group: any) {
        const { members = [] } = cfg
        
        // 绘制虚线边框
        const rect = group.addShape('rect', {
          attrs: {
            x: -80,
            y: -60,
            width: 160,
            height: 120,
            stroke: '#ffd700',
            lineDash: [5, 5],
            lineWidth: 2,
            radius: 8,
            fill: 'rgba(255, 215, 0, 0.05)'
          },
          name: 'committee-border'
        })
        
        // 委员会标签
        group.addShape('text', {
          attrs: {
            x: 0,
            y: -70,
            text: cfg.label,
            fontSize: 14,
            fill: '#ffd700',
            textAlign: 'center',
            textBaseline: 'bottom',
            fontWeight: 600
          },
          name: 'committee-label'
        })
        
        // 绘制4个成员节点
        const positions = [
          { x: -40, y: -20 },
          { x: 40, y: -20 },
          { x: -40, y: 20 },
          { x: 40, y: 20 }
        ]
        
        members.forEach((member: any, i: number) => {
          const pos = positions[i] || { x: 0, y: 0 }
          const isLeader = member.isLeader
          
          // 成员节点圆形
          group.addShape('circle', {
            attrs: {
              x: pos.x,
              y: pos.y,
              r: 12,
              fill: isLeader ? '#ffd700' : '#52c41a',
              stroke: '#fff',
              lineWidth: 2
            },
            name: `member-${i}`
          })
          
          // Leader 星标
          if (isLeader) {
            group.addShape('text', {
              attrs: {
                x: pos.x,
                y: pos.y + 4,
                text: '★',
                fontSize: 14,
                fill: '#fff',
                textAlign: 'center',
                textBaseline: 'middle'
              },
              name: `leader-star-${i}`
            })
          }
        })
        
        return rect
      }
    }, 'rect')
    
    // 注册POT节点
    G6.registerNode('pot-node', {
      draw(cfg: any, group: any) {
        const { status } = cfg
        const color = '#00d4ff'
        const opacity = status === 'offline' ? 0.4 : 1
        
        // 在线节点光晕
        if (status === 'online') {
          group.addShape('circle', {
            attrs: {
              x: 0,
              y: 0,
              r: 20,
              fill: color,
              opacity: 0.2
            },
            name: 'glow'
          })
        }
        
        // 主圆形
        const mainCircle = group.addShape('circle', {
          attrs: {
            x: 0,
            y: 0,
            r: 14,
            fill: color,
            stroke: '#fff',
            lineWidth: 2,
            opacity,
            cursor: 'pointer'
          },
          name: 'main-circle'
        })
        
        // 节点标签
        group.addShape('text', {
          attrs: {
            x: 0,
            y: 28,
            text: cfg.label,
            fontSize: 11,
            fill: '#fff',
            textAlign: 'center',
            textBaseline: 'top'
          },
          name: 'label'
        })
        
        return mainCircle
      }
    }, 'circle')
    
    // 创建图实例
    graph = new G6.Graph({
      container: container.value!,
      width: container.value!.clientWidth,
      height: container.value!.clientHeight,
      modes: {
        default: ['drag-canvas', 'zoom-canvas']
      },
      defaultNode: {
        type: 'pot-node',
        size: 28
      },
      defaultEdge: {
        type: 'line',
        style: {
          stroke: 'rgba(255, 255, 255, 0.15)',
          lineWidth: 1,
          endArrow: false
        }
      },
      nodeStateStyles: {
        hover: {
          stroke: '#ffd700',
          lineWidth: 3
        }
      },
      edgeStateStyles: {
        hover: {
          stroke: 'rgba(0, 212, 255, 0.8)',
          lineWidth: 2
        }
      }
    })
    
    // 节点点击事件
    graph.on('node:click', (evt: any) => {
      const node = evt.item
      const model = node.getModel()
      if (model.type !== 'committee-group') {
        selectedNode.value = model
        detailVisible.value = true
      }
    })
    
    // 节点悬停效果
    graph.on('node:mouseenter', (evt: any) => {
      const node = evt.item
      graph.setItemState(node, 'hover', true)
    })
    
    graph.on('node:mouseleave', (evt: any) => {
      const node = evt.item
      graph.setItemState(node, 'hover', false)
    })
    
    // 初始渲染
    updateGraph()
  }).catch((err) => {
    console.warn('G6 not installed yet. Run npm install to use network topology.', err)
  })
}

function updateGraph() {
  if (!graph || !topology.value) return
  
  // 生成委员会和POT节点的双层结构
  const nodes = []
  const edges = []
  
  // 上层：委员会组（4个）
  const committeeCount = 4
  const committeeY = -200 // 上层Y坐标
  const committeeSpacing = 300
  const startX = -(committeeCount - 1) * committeeSpacing / 2
  
  for (let i = 0; i < committeeCount; i++) {
    const members = []
    for (let j = 0; j < 4; j++) {
      members.push({
        id: `c${i}-m${j}`,
        isLeader: j === 0 // 第一个是Leader
      })
    }
    
    nodes.push({
      id: `committee-${i}`,
      label: `委员会 ${i + 1}`,
      type: 'committee-group',
      x: startX + i * committeeSpacing,
      y: committeeY,
      members
    })
  }
  
  // 下层：POT节点集群
  const potNodes = topology.value.nodes.filter(n => n.type === 'pot' || !n.type)
  const potY = 100 // 下层Y坐标
  const potNodesPerRow = 8
  const potSpacing = 100
  
  potNodes.forEach((node, index) => {
    const row = Math.floor(index / potNodesPerRow)
    const col = index % potNodesPerRow
    const rowStartX = -(potNodesPerRow - 1) * potSpacing / 2
    
    nodes.push({
      id: String(node.id),
      label: `Node ${node.id}`,
      type: 'pot-node',
      status: node.status || 'online',
      address: node.address,
      connections: node.connections,
      latency: node.latency,
      x: rowStartX + col * potSpacing,
      y: potY + row * 80
    })
  })
  
  // 添加边：委员会之间的连接
  for (let i = 0; i < committeeCount - 1; i++) {
    edges.push({
      source: `committee-${i}`,
      target: `committee-${i + 1}`
    })
  }
  
  // 添加边：POT节点之间的连接（部分）
  topology.value.edges.forEach(edge => {
    const sourceExists = potNodes.find(n => String(n.id) === String(edge.source))
    const targetExists = potNodes.find(n => String(n.id) === String(edge.target))
    if (sourceExists && targetExists) {
      edges.push({
        source: String(edge.source),
        target: String(edge.target)
      })
    }
  })
  
  // 添加边：委员会到POT节点的连接（每个委员会连接到几个POT节点）
  potNodes.slice(0, 12).forEach((node, index) => {
    const committeeIndex = index % committeeCount
    edges.push({
      source: `committee-${committeeIndex}`,
      target: String(node.id),
      style: {
        stroke: 'rgba(255, 215, 0, 0.2)',
        lineDash: [3, 3]
      }
    })
  })
  
  graph.data({ nodes, edges })
  graph.render()
  
  // 适应画布
  setTimeout(() => {
    graph.fitView(40)
  }, 300)
}

function zoomIn() {
  if (graph) {
    const currentZoom = graph.getZoom()
    graph.zoomTo(currentZoom * 1.2)
  }
}

function zoomOut() {
  if (graph) {
    const currentZoom = graph.getZoom()
    graph.zoomTo(currentZoom * 0.8)
  }
}

function fitView() {
  if (graph) {
    graph.fitView(20)
  }
}

function getNodeTypeColor(type: string) {
  const colors: Record<string, string> = {
    'leader': 'gold',
    'committee': 'green',
    'pot': 'blue'
  }
  return colors[type] || 'default'
}

function getNodeTypeName(type: string) {
  const names: Record<string, string> = {
    'leader': 'Leader',
    'committee': '委员会成员',
    'pot': 'POT节点'
  }
  return names[type] || type
}

function getStatusType(status: string) {
  const map: Record<string, any> = {
    'online': 'success',
    'offline': 'default',
    'error': 'error'
  }
  return map[status] || 'default'
}

watch(() => topology.value, () => {
  updateGraph()
}, { deep: true })

onMounted(() => {
  initGraph()
})
</script>

<style scoped>
.topology-container {
  background: linear-gradient(135deg, #0a0e27 0%, #141d3a 100%);
  border-radius: 8px;
  position: relative;
}

.legend {
  backdrop-filter: blur(10px);
}

.space-y-2 > * + * {
  margin-top: 0.5rem;
}

.space-y-3 > * + * {
  margin-top: 0.75rem;
}
</style>
