<template>
  <div class="network-topology h-full relative">
    <div class="flex items-center justify-between mb-4">
      <h3 class="text-lg font-semibold flex items-center gap-2">
        <NodeIndexOutlined />
        网络拓扑
      </h3>
      
      <!-- 工具栏 -->
      <div class="flex items-center gap-2">
        <a-tooltip title="重置视角">
          <a-button size="small" @click="resetCamera">
            <template #icon><FullscreenOutlined /></template>
          </a-button>
        </a-tooltip>
      </div>
    </div>
    
    <!-- Three.js 容器 -->
    <div ref="container" class="topology-container" style="height: calc(100% - 60px);"></div>
    
    <!-- 图例 -->
    <div class="legend absolute bottom-4 right-4 bg-bg-primary bg-opacity-80 p-3 rounded border border-border">
      <div class="text-xs space-y-2">
        <div class="flex items-center gap-2">
          <span class="w-3 h-3 rounded-full bg-orange-500"></span>
          <span>委员会领导者</span>
        </div>
        <div class="flex items-center gap-2">
          <span class="w-3 h-3 rounded-full bg-green-500"></span>
          <span>委员会成员</span>
        </div>
        <div class="flex items-center gap-2">
          <span class="w-3 h-3 rounded-full bg-blue-500"></span>
          <span>POT节点</span>
        </div>
        <div class="flex items-center gap-2">
          <span class="w-3 h-3 rounded-full bg-blue-400 animate-pulse"></span>
          <span>本机节点</span>
        </div>
        <!-- <div class="flex items-center gap-1 mt-2">
          <div class="w-8 h-0.5 bg-white opacity-30"></div>
          <span>层内连接</span>
        </div>
        <div class="flex items-center gap-1">
          <div class="w-8 h-0.5 border-t border-dashed border-white opacity-30"></div>
          <span>跨层连接</span>
        </div> -->
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
          <a-descriptions-item label="类型">
            <a-tag :color="getNodeTypeColor(selectedNode.layer, selectedNode.isLeader, selectedNode.isLocal)">
              {{ getNodeTypeName(selectedNode.layer, selectedNode.isLeader, selectedNode.isLocal) }}
            </a-tag>
          </a-descriptions-item>
          <a-descriptions-item label="层级">
            {{ selectedNode.layer === 'committee' ? '委员会层' : 'POT节点层' }}
          </a-descriptions-item>
          <a-descriptions-item v-if="selectedNode.isLeader" label="角色">
            <a-tag color="orange">领导者</a-tag>
          </a-descriptions-item>
          <a-descriptions-item v-if="selectedNode.isLocal" label="本机">
            <a-badge status="processing" text="是" />
          </a-descriptions-item>
        </a-descriptions>
      </div>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, watch } from 'vue'
import { storeToRefs } from 'pinia'
import {
  NodeIndexOutlined,
  FullscreenOutlined
} from '@ant-design/icons-vue'
import { useNetworkStore } from '@/stores/network'
import * as THREE from 'three'
import { OrbitControls } from 'three/addons/controls/OrbitControls.js'

const container = ref<HTMLDivElement>()
const networkStore = useNetworkStore()
const { topology } = storeToRefs(networkStore)

const detailVisible = ref(false)
const selectedNode = ref<any>(null)

let scene: THREE.Scene
let camera: THREE.PerspectiveCamera
let renderer: THREE.WebGLRenderer
let controls: OrbitControls
let animationId: number
let nodeObjects: Array<{ mesh: THREE.Mesh; data: any; pulsePhase: number }> = []

// 节点配置
const NODES_PER_ROW = 6 // 每行节点数量
const NODES_PER_COL = 4 // 每列节点数量
const NODE_COUNT = NODES_PER_ROW * NODES_PER_COL // 总节点数
const LAYER_SPACING = 3 // 层间距
const POT_Y = 0 // 底层 Y 坐标
const COMMITTEE_Y = POT_Y + LAYER_SPACING // 上层 Y 坐标

function initThreeJS() {
  if (!container.value) return

  const width = container.value.clientWidth
  const height = container.value.clientHeight

  // 创建场景
  scene = new THREE.Scene()
  scene.fog = new THREE.Fog(0x0a0e27, 10, 50)

  // 创建相机
  camera = new THREE.PerspectiveCamera(60, width / height, 0.1, 1000)
  camera.position.set(8, 10, 12)
  camera.lookAt(0, 1.5, 0)

  // 创建渲染器
  renderer = new THREE.WebGLRenderer({ antialias: true, alpha: true })
  renderer.setSize(width, height)
  renderer.setPixelRatio(window.devicePixelRatio)
  renderer.setClearColor(0x0a0e27, 0)
  container.value.appendChild(renderer.domElement)

  // 添加控制器
  controls = new OrbitControls(camera, renderer.domElement)
  controls.enableDamping = true
  controls.dampingFactor = 0.05
  controls.minDistance = 5
  controls.maxDistance = 30
  controls.maxPolarAngle = Math.PI / 2.2

  // 添加光源
  const ambientLight = new THREE.AmbientLight(0xffffff, 0.5)
  scene.add(ambientLight)

  const pointLight1 = new THREE.PointLight(0x00d4ff, 1, 50)
  pointLight1.position.set(10, 10, 10)
  scene.add(pointLight1)

  const pointLight2 = new THREE.PointLight(0xff6b35, 0.8, 50)
  pointLight2.position.set(-10, 10, -10)
  scene.add(pointLight2)

  // 创建网络拓扑
  createNetworkTopology()

  // 添加点击事件
  renderer.domElement.addEventListener('click', onCanvasClick)

  // 窗口大小调整
  window.addEventListener('resize', onWindowResize)

  // 开始动画
  animate()
}

function createNetworkTopology() {
  nodeObjects = []

  // 随机分布的节点位置（带最小距离约束）
  const areaSize = 10 // 分布区域大小
  const nodeCount = NODES_PER_ROW * NODES_PER_COL
  const minDistance = 1.2 // 节点之间的最小距离
  
  // 存储节点位置用于层内连接
  const nodePositions: Array<{ x: number; z: number; index: number }> = []

  // 生成随机位置，确保节点之间保持最小距离
  for (let i = 0; i < nodeCount; i++) {
    let x: number, z: number
    let attempts = 0
    const maxAttempts = 100
    
    do {
      x = (Math.random() - 0.5) * areaSize
      z = (Math.random() - 0.5) * areaSize
      attempts++
      
      // 检查与已有节点的距离
      const tooClose = nodePositions.some(pos => {
        const distance = Math.sqrt(
          Math.pow(x - pos.x, 2) + Math.pow(z - pos.z, 2)
        )
        return distance < minDistance
      })
      
      if (!tooClose || attempts >= maxAttempts) {
        break
      }
    } while (true)
    
    nodePositions.push({ x, z, index: i })
  }

  // === 底层：POT 节点（使用生成的位置）===
  nodePositions.forEach((pos, i) => {
    const isLocal = i === 5 // 第6个节点是本机节点

    // 创建节点球体
    const geometry = new THREE.SphereGeometry(0.2, 32, 32)
    const material = new THREE.MeshStandardMaterial({
      color: 0x3b82f6, // 蓝色
      emissive: isLocal ? 0x3b82f6 : 0x1e40af,
      emissiveIntensity: isLocal ? 0.5 : 0.2,
      metalness: 0.3,
      roughness: 0.4
    })
    const nodeMesh = new THREE.Mesh(geometry, material)
    nodeMesh.position.set(pos.x, POT_Y, pos.z)
    scene.add(nodeMesh)

    // 添加光晕效果（本机节点）
    if (isLocal) {
      const glowGeometry = new THREE.SphereGeometry(0.3, 32, 32)
      const glowMaterial = new THREE.MeshBasicMaterial({
        color: 0x3b82f6,
        transparent: true,
        opacity: 0.3
      })
      const glow = new THREE.Mesh(glowGeometry, glowMaterial)
      nodeMesh.add(glow)
    }

    nodeObjects.push({
      mesh: nodeMesh,
      data: {
        id: `pot-${i}`,
        layer: 'pot',
        index: i,
        isLocal,
        isLeader: false
      },
      pulsePhase: isLocal ? Math.random() * Math.PI * 2 : 0
    })
  })

  // === 上层：委员会节点（与底层节点位置一一对应）===
  nodePositions.forEach((pos, i) => {
    const isLeader = i === 0 // 第一个节点是领导者

    // 创建节点球体（使用相同的x,z坐标，只改变y坐标）
    const geometry = new THREE.SphereGeometry(0.25, 32, 32)
    const material = new THREE.MeshStandardMaterial({
      color: isLeader ? 0xf97316 : 0x22c55e, // 橙色/绿色
      emissive: isLeader ? 0xf97316 : 0x22c55e,
      emissiveIntensity: isLeader ? 0.6 : 0.3,
      metalness: 0.4,
      roughness: 0.3
    })
    const nodeMesh = new THREE.Mesh(geometry, material)
    nodeMesh.position.set(pos.x, COMMITTEE_Y, pos.z)
    scene.add(nodeMesh)

    // 领导者添加外圈标识
    if (isLeader) {
      const ringGeometry = new THREE.TorusGeometry(0.35, 0.03, 16, 32)
      const ringMaterial = new THREE.MeshBasicMaterial({
        color: 0xfbbf24,
        transparent: true,
        opacity: 0.8
      })
      const ring = new THREE.Mesh(ringGeometry, ringMaterial)
      ring.rotation.x = Math.PI / 2
      nodeMesh.add(ring)
    }

    nodeObjects.push({
      mesh: nodeMesh,
      data: {
        id: `committee-${i}`,
        layer: 'committee',
        index: i,
        isLocal: false,
        isLeader
      },
      pulsePhase: 0
    })
  })

  // === 层内连接（细实线 - 点对点连接附近节点）===
  // POT 层内连接 - 确保每个节点至少有一个连接
  const connectionDistance = 3 // 连接距离阈值
  const potConnected = new Set<number>() // 记录已连接的节点
  
  // 第一遍：连接距离较近的节点
  for (let i = 0; i < nodePositions.length; i++) {
    for (let j = i + 1; j < nodePositions.length; j++) {
      const pos1 = nodePositions[i]
      const pos2 = nodePositions[j]
      const distance = Math.sqrt(
        Math.pow(pos1.x - pos2.x, 2) + Math.pow(pos1.z - pos2.z, 2)
      )
      
      // 只连接距离较近的节点
      if (distance < connectionDistance) {
        createLine(
          new THREE.Vector3(pos1.x, POT_Y, pos1.z),
          new THREE.Vector3(pos2.x, POT_Y, pos2.z),
          0x3b82f6,
          0.02,
          false
        )
        potConnected.add(i)
        potConnected.add(j)
      }
    }
  }
  
  // 第二遍：为未连接的节点找到最近的节点进行连接
  for (let i = 0; i < nodePositions.length; i++) {
    if (!potConnected.has(i)) {
      let minDist = Infinity
      let nearestIdx = -1
      
      // 找到最近的节点
      for (let j = 0; j < nodePositions.length; j++) {
        if (i === j) continue
        const pos1 = nodePositions[i]
        const pos2 = nodePositions[j]
        const distance = Math.sqrt(
          Math.pow(pos1.x - pos2.x, 2) + Math.pow(pos1.z - pos2.z, 2)
        )
        if (distance < minDist) {
          minDist = distance
          nearestIdx = j
        }
      }
      
      // 连接到最近的节点
      if (nearestIdx !== -1) {
        const pos1 = nodePositions[i]
        const pos2 = nodePositions[nearestIdx]
        createLine(
          new THREE.Vector3(pos1.x, POT_Y, pos1.z),
          new THREE.Vector3(pos2.x, POT_Y, pos2.z),
          0x3b82f6,
          0.02,
          false
        )
        potConnected.add(i)
      }
    }
  }

  // 委员会层内连接 - 确保每个节点至少有一个连接（使用相同的位置关系）
  const committeeConnected = new Set<number>()
  
  // 第一遍：连接距离较近的节点
  for (let i = 0; i < nodePositions.length; i++) {
    for (let j = i + 1; j < nodePositions.length; j++) {
      const pos1 = nodePositions[i]
      const pos2 = nodePositions[j]
      const distance = Math.sqrt(
        Math.pow(pos1.x - pos2.x, 2) + Math.pow(pos1.z - pos2.z, 2)
      )
      
      // 只连接距离较近的节点
      if (distance < connectionDistance) {
        createLine(
          new THREE.Vector3(pos1.x, COMMITTEE_Y, pos1.z),
          new THREE.Vector3(pos2.x, COMMITTEE_Y, pos2.z),
          0x22c55e,
          0.02,
          false
        )
        committeeConnected.add(i)
        committeeConnected.add(j)
      }
    }
  }
  
  // 第二遍：为未连接的节点找到最近的节点进行连接
  for (let i = 0; i < nodePositions.length; i++) {
    if (!committeeConnected.has(i)) {
      let minDist = Infinity
      let nearestIdx = -1
      
      // 找到最近的节点
      for (let j = 0; j < nodePositions.length; j++) {
        if (i === j) continue
        const pos1 = nodePositions[i]
        const pos2 = nodePositions[j]
        const distance = Math.sqrt(
          Math.pow(pos1.x - pos2.x, 2) + Math.pow(pos1.z - pos2.z, 2)
        )
        if (distance < minDist) {
          minDist = distance
          nearestIdx = j
        }
      }
      
      // 连接到最近的节点
      if (nearestIdx !== -1) {
        const pos1 = nodePositions[i]
        const pos2 = nodePositions[nearestIdx]
        createLine(
          new THREE.Vector3(pos1.x, COMMITTEE_Y, pos1.z),
          new THREE.Vector3(pos2.x, COMMITTEE_Y, pos2.z),
          0x22c55e,
          0.02,
          false
        )
        committeeConnected.add(i)
      }
    }
  }

  // === 跨层连接（虚线 - 垂直连接对应节点）===
  for (let i = 0; i < nodePositions.length; i++) {
    const pos = nodePositions[i]
    
    createLine(
      new THREE.Vector3(pos.x, POT_Y, pos.z),
      new THREE.Vector3(pos.x, COMMITTEE_Y, pos.z),
      0x64748b,
      0.015,
      true // 虚线
    )
  }

  // === 创建平面网格（增强 3D 感）===
  const gridSize = areaSize * 1.3
  const gridHelper = new THREE.GridHelper(gridSize, 20, 0x334155, 0x1e293b)
  gridHelper.position.y = POT_Y - 0.5
  gridHelper.material.opacity = 0.3
  gridHelper.material.transparent = true
  scene.add(gridHelper)
}

function createLine(
  start: THREE.Vector3,
  end: THREE.Vector3,
  color: number,
  lineWidth: number,
  dashed: boolean
) {
  const points = [start, end]
  const geometry = new THREE.BufferGeometry().setFromPoints(points)
  
  let material
  if (dashed) {
    material = new THREE.LineDashedMaterial({
      color,
      linewidth: lineWidth,
      dashSize: 0.2,
      gapSize: 0.15,
      transparent: true,
      opacity: 0.4
    })
  } else {
    material = new THREE.LineBasicMaterial({
      color,
      linewidth: lineWidth,
      transparent: true,
      opacity: 0.3
    })
  }
  
  const line = new THREE.Line(geometry, material)
  if (dashed) {
    line.computeLineDistances()
  }
  scene.add(line)
}

function animate() {
  animationId = requestAnimationFrame(animate)

  // 更新本机节点的脉冲效果
  nodeObjects.forEach((obj) => {
    if (obj.data.isLocal) {
      obj.pulsePhase += 0.05
      const scale = 1 + Math.sin(obj.pulsePhase) * 0.15
      obj.mesh.scale.set(scale, scale, scale)
      
      // 更新发光强度
      const material = obj.mesh.material as THREE.MeshStandardMaterial
      material.emissiveIntensity = 0.5 + Math.sin(obj.pulsePhase) * 0.3
    }
  })

  controls.update()
  renderer.render(scene, camera)
}

function onCanvasClick(event: MouseEvent) {
  const rect = container.value?.getBoundingClientRect()
  if (!rect) return

  const mouse = new THREE.Vector2()
  mouse.x = ((event.clientX - rect.left) / rect.width) * 2 - 1
  mouse.y = -((event.clientY - rect.top) / rect.height) * 2 + 1

  const raycaster = new THREE.Raycaster()
  raycaster.setFromCamera(mouse, camera)

  const meshes = nodeObjects.map(obj => obj.mesh)
  const intersects = raycaster.intersectObjects(meshes)

  if (intersects.length > 0) {
    const clickedMesh = intersects[0].object as THREE.Mesh
    const nodeObj = nodeObjects.find(obj => obj.mesh === clickedMesh)
    if (nodeObj) {
      selectedNode.value = nodeObj.data
      detailVisible.value = true
    }
  }
}

function onWindowResize() {
  if (!container.value) return
  
  const width = container.value.clientWidth
  const height = container.value.clientHeight

  camera.aspect = width / height
  camera.updateProjectionMatrix()
  renderer.setSize(width, height)
}

function resetCamera() {
  camera.position.set(8, 10, 12)
  camera.lookAt(0, 1.5, 0)
  controls.target.set(0, 1.5, 0)
  controls.update()
}

function getNodeTypeColor(layer: string, isLeader: boolean, isLocal: boolean) {
  if (isLocal) return 'blue'
  if (layer === 'committee') {
    return isLeader ? 'orange' : 'green'
  }
  return 'blue'
}

function getNodeTypeName(layer: string, isLeader: boolean, isLocal: boolean) {
  if (isLocal) return '本机POT节点'
  if (layer === 'committee') {
    return isLeader ? '委员会领导者' : '委员会成员'
  }
  return 'POT节点'
}

watch(() => topology.value, () => {
  // 如果需要根据实时数据更新，可以在这里重新创建拓扑
}, { deep: true })

onMounted(() => {
  initThreeJS()
})

onUnmounted(() => {
  if (animationId) {
    cancelAnimationFrame(animationId)
  }
  if (renderer) {
    renderer.dispose()
  }
  if (container.value && renderer) {
    container.value.removeChild(renderer.domElement)
  }
  window.removeEventListener('resize', onWindowResize)
})
</script>

<style scoped>
.topology-container {
  background: transparent;
  border-radius: 8px;
  position: relative;
  overflow: hidden;
}

.topology-container :deep(canvas) {
  display: block;
}

.legend {
  backdrop-filter: blur(10px);
  background: rgba(10, 14, 39, 0.8) !important;
  z-index: 10;
}

/* 半透明按钮样式 */
:deep(.ant-btn) {
  background: rgba(255, 255, 255, 0.1) !important;
  border-color: rgba(255, 255, 255, 0.2) !important;
  color: rgba(255, 255, 255, 0.8) !important;
  backdrop-filter: blur(5px);
}

:deep(.ant-btn:hover) {
  background: rgba(255, 255, 255, 0.2) !important;
  border-color: rgba(0, 212, 255, 0.5) !important;
  color: #00d4ff !important;
}

.space-y-2 > * + * {
  margin-top: 0.5rem;
}

.space-y-3 > * + * {
  margin-top: 0.75rem;
}

@keyframes pulse {
  0%, 100% {
    opacity: 1;
  }
  50% {
    opacity: 0.5;
  }
}

.animate-pulse {
  animation: pulse 2s cubic-bezier(0.4, 0, 0.6, 1) infinite;
}
</style>
