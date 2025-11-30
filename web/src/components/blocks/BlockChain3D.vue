<template>
  <div class="blockchain-3d-wrapper">
    <div class="blockchain-3d-container" ref="containerRef"></div>
    
    <!-- 区块详情模态框 -->
    <a-modal 
      v-model:visible="showBlockDetail" 
      title="区块详情" 
      :footer="null"
      :width="600"
    >
      <div v-if="selectedBlock" class="space-y-3">
        <div class="flex justify-between items-center">
          <span class="text-gray-400">区块高度</span>
          <span class="text-xl font-bold text-blue-400">#{{ selectedBlock.height }}</span>
        </div>
        <a-divider style="margin: 12px 0" />
        <div class="flex justify-between">
          <span class="text-gray-400">区块哈希</span>
          <span class="font-mono text-sm">{{ selectedBlock.hash }}</span>
        </div>
        <div class="flex justify-between">
          <span class="text-gray-400">时间戳</span>
          <span>{{ formatTimestamp(selectedBlock.timestamp) }}</span>
        </div>
        <div class="flex justify-between">
          <span class="text-gray-400">交易数量</span>
          <span>{{ selectedBlock.txCount }}</span>
        </div>
        <div class="flex justify-between">
          <span class="text-gray-400">区块大小</span>
          <span>{{ formatSize(selectedBlock.size) }}</span>
        </div>
        <div class="flex justify-between">
          <span class="text-gray-400">矿工</span>
          <span class="font-mono text-xs">{{ selectedBlock.miner }}</span>
        </div>
      </div>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import * as THREE from 'three'
import { api } from '@/services/api'
import type { BlockInfo } from '@/types/api'

const containerRef = ref<HTMLElement | null>(null)
const showBlockDetail = ref(false)
const selectedBlock = ref<BlockInfo & {
  hash: string
  timestamp: number
  txCount: number
  size: number
  miner: string
} | null>(null)

let scene: THREE.Scene | null = null
let camera: THREE.OrthographicCamera | null = null
let renderer: THREE.WebGLRenderer | null = null
let animationId: number | null = null
let raycaster: THREE.Raycaster | null = null
let mouse: THREE.Vector2 | null = null

interface Block {
  mesh: THREE.Group
  arrow?: THREE.Group
  height: number
  hash: string
  position: number
  timestamp: number
  txCount: number
  size: number
  miner: string
}

const blocks: Block[] = []
const blockSpacing = 5  // 减小区块间距，使更多区块能在屏幕上显示
let maxBlocks = 15  // 改为let以便动态更新
let nextPosition = 0
let initialOffset = 0  // 初始偏移量，用于将第一个区块放在左侧
let latestBlockHeight = 0  // 记录最新区块高度
let pollInterval: number | null = null  // 轮询定时器

// 格式化时间戳
function formatTimestamp(timestamp: number): string {
  return new Date(timestamp).toLocaleString('zh-CN')
}

// 格式化大小
function formatSize(size: number): string {
  if (size < 1024) return size + ' B'
  if (size < 1024 * 1024) return (size / 1024).toFixed(2) + ' KB'
  return (size / (1024 * 1024)).toFixed(2) + ' MB'
}

onMounted(() => {
  if (!containerRef.value) return
  
  // 延迟初始化，确保容器尺寸已计算完成
  setTimeout(() => {
    initScene()
    initBlocksFromBackend()
    animate()
    
    // 定期检查新区块（每3秒）
    pollInterval = window.setInterval(() => {
      checkForNewBlocks()
    }, 3000)
  }, 100)
  
  onUnmounted(() => {
    if (pollInterval !== null) {
      clearInterval(pollInterval)
    }
    cleanup()
  })
  
  window.addEventListener('resize', handleResize)
})

function initScene() {
  if (!containerRef.value) return
  
  const width = containerRef.value.clientWidth
  const height = containerRef.value.clientHeight
  
  // 验证容器尺寸 - 如果无效，延迟重试
  if (width <= 0 || height <= 0) {
    console.warn('[BlockChain3D] Invalid container size, retrying in 200ms:', { width, height })
    setTimeout(() => initScene(), 200)
    return
  }
  
  console.log('[BlockChain3D] Initializing scene with size:', { width, height })
  
  // 场景
  scene = new THREE.Scene()
  
  // 正交相机 - 保持二维视角，无透视变形
  const aspect = width / height
  const viewSize = 4  // 减小viewSize使区块看起来更大
  camera = new THREE.OrthographicCamera(
    -viewSize * aspect,
    viewSize * aspect,
    viewSize,
    -viewSize,
    0.1,
    1000
  )
  camera.position.set(0, 3, 6)  // 调整相机位置，更接近场景
  camera.up.set(0, 1, 0)  // 确保Y轴为上方向
  camera.lookAt(0, 0, 0)
  
  // 根据屏幕宽度计算最大块数
  // 相机可见宽度 = 2 * viewSize * aspect
  // 每个区块占用空间 = blockSpacing (包括区块本身和箭头)
  // 首先检查aspect是否有效
  if (!isFinite(aspect) || aspect <= 0) {
    console.warn('[BlockChain3D] Invalid aspect:', aspect, '- using default values')
    maxBlocks = 10
    initialOffset = -20
  } else {
    const visibleWidth = 2 * viewSize * aspect
    if (!isFinite(visibleWidth) || visibleWidth <= 0) {
      console.warn('[BlockChain3D] Invalid visibleWidth:', visibleWidth, '- using default maxBlocks')
      maxBlocks = 10
      initialOffset = -20
    } else {
      maxBlocks = Math.max(6, Math.floor(visibleWidth / blockSpacing))
      // 计算初始偏移：使第一个区块从左侧边缘开始
      initialOffset = -visibleWidth / 2 + blockSpacing / 2
    }
  }
  
  const visibleWidthStr = isFinite(2 * viewSize * aspect) ? (2 * viewSize * aspect).toFixed(2) : 'Infinity'
  const initialOffsetStr = isFinite(initialOffset) ? initialOffset.toFixed(2) : 'Invalid'
  console.log(`[BlockChain3D] 屏幕宽度: ${width}px, 高度: ${height}px, aspect: ${aspect.toFixed(2)}, 视口宽度: ${visibleWidthStr}, 最大块数: ${maxBlocks}, 初始偏移: ${initialOffsetStr}`)
  
  // 渲染器
  renderer = new THREE.WebGLRenderer({ 
    alpha: true, 
    antialias: true 
  })
  renderer.setSize(width, height)
  renderer.setPixelRatio(window.devicePixelRatio)
  renderer.setClearColor(0x000000, 0)
  containerRef.value.appendChild(renderer.domElement)
  
  // 光源
  const ambientLight = new THREE.AmbientLight(0xffffff, 0.6)
  scene.add(ambientLight)
  
  const directionalLight = new THREE.DirectionalLight(0xffffff, 0.8)
  directionalLight.position.set(10, 10, 10)
  directionalLight.castShadow = true
  scene.add(directionalLight)
  
  // 多个点光源增强立体感
  const pointLight1 = new THREE.PointLight(0x00d4ff, 1.5, 30)
  pointLight1.position.set(-5, 3, 5)
  scene.add(pointLight1)
  
  const pointLight2 = new THREE.PointLight(0xffd700, 1, 30)
  pointLight2.position.set(10, 3, 5)
  scene.add(pointLight2)
  
  // 初始化raycaster用于检测点击
  raycaster = new THREE.Raycaster()
  mouse = new THREE.Vector2()
  
  // 添加鼠标事件监听
  renderer.domElement.addEventListener('click', onBlockClick)
  renderer.domElement.addEventListener('mousemove', onMouseMove)
  renderer.domElement.style.cursor = 'default'
}

// 从后端初始化区块
async function initBlocksFromBackend() {
  try {
    console.log('[BlockChain3D] Fetching recent blocks from backend...')
    // 获取最近的maxBlocks个区块
    const recentBlocks = await api.getRecentBlocks(maxBlocks)
    
    console.log('[BlockChain3D] Received blocks:', recentBlocks)
    
    if (recentBlocks.length === 0) {
      console.warn('[BlockChain3D] No blocks returned from backend, will retry via polling')
      // 不返回，让轮询机制继续尝试获取
      return
    }
    
    // 按高度升序排列
    recentBlocks.sort((a, b) => a.height - b.height)
    
    // 添加所有区块（不触发场景移动）
    for (const blockInfo of recentBlocks) {
      console.log('[BlockChain3D] Adding block:', blockInfo.height)
      addBlock(
        blockInfo.height,
        blockInfo.hash,
        false,
        blockInfo.timestamp * 1000, // 转换为毫秒
        blockInfo
      )
    }
    
    // 记录最新区块高度
    if (recentBlocks.length > 0) {
      latestBlockHeight = recentBlocks[recentBlocks.length - 1].height
    }
    
    // 确保场景位置为原点
    if (scene) {
      scene.position.x = 0
    }
    
    console.log(`[BlockChain3D] Initialized ${recentBlocks.length} blocks, latest height: ${latestBlockHeight}`)
  } catch (error) {
    console.error('[BlockChain3D] Failed to initialize blocks from backend:', error)
  }
}

// 检查是否有新区块
async function checkForNewBlocks() {
  try {
    // 获取最新的一个区块
    const recentBlocks = await api.getRecentBlocks(1)
    
    if (recentBlocks.length === 0) {
      return
    }
    
    const latestBlock = recentBlocks[0]
    
    // 如果发现新区块，添加到场景中
    if (latestBlock.height > latestBlockHeight) {
      console.log(`[BlockChain3D] New block detected: height ${latestBlock.height}`)
      addBlock(
        latestBlock.height,
        latestBlock.hash,
        true, // 标记为新区块
        latestBlock.timestamp * 1000,
        latestBlock
      )
      latestBlockHeight = latestBlock.height
    }
  } catch (error) {
    console.error('[BlockChain3D] Failed to check for new blocks:', error)
  }
}

function createBlockMesh(height: number, hash: string, isNew: boolean): THREE.Group {
  const group = new THREE.Group()
  
  // 设置固定旋转角度，显示三个面
  group.rotation.y = Math.PI / 6  // 30度
  group.rotation.x = - Math.PI / 12 // 15度
  
  // 正方体几何体
  const size = 3  // 减小尺寸以适应更多区块
  const geometry = new THREE.BoxGeometry(size, size, size)
  
  // 材质 - 第一个区块为红色，其他为蓝色
  let color: THREE.Color
  const blockIndex = blocks.length
  if (blockIndex === 0) {
    // 第一个区块为红色
    color = new THREE.Color(0xff6b6b)
  } else {
    // 其他区块统一为蓝色
    color = new THREE.Color(0x00d4ff)
  }
  
  const material = new THREE.MeshStandardMaterial({
    color: color,
    emissive: isNew ? color : new THREE.Color(0x002244),
    emissiveIntensity: isNew ? 0.6 : 0.15,
    metalness: 0.7,
    roughness: 0.3,
    transparent: true,
    opacity: 0.95
  })
  
  const cube = new THREE.Mesh(geometry, material)
  cube.castShadow = true
  cube.receiveShadow = true
  group.add(cube)
  
  // 边框线
  const edges = new THREE.EdgesGeometry(geometry)
  const lineMaterial = new THREE.LineBasicMaterial({ 
    color: 0xffffff, 
    transparent: true, 
    opacity: 0.8,
    linewidth: 2
  })
  const wireframe = new THREE.LineSegments(edges, lineMaterial)
  group.add(wireframe)
  
  // 创建文字纹理（多面显示）
  const canvas = document.createElement('canvas')
  const context = canvas.getContext('2d')!
  canvas.width = 512
  canvas.height = 512
  
  // 背景渐变
  const gradient = context.createLinearGradient(0, 0, 0, 512)
  gradient.addColorStop(0, 'rgba(0, 212, 255, 0.1)')
  gradient.addColorStop(1, 'rgba(0, 100, 150, 0.1)')
  context.fillStyle = gradient
  context.fillRect(0, 0, 512, 512)
  
  // 绘制区块高度
  context.fillStyle = '#ffffff'
  context.font = 'bold 64px Arial'
  context.textAlign = 'center'
  context.textBaseline = 'middle'
  context.fillText(`#${height}`, 256, 180)
  
  // 绘制哈希
  context.font = 'bold 28px monospace'
  context.fillStyle = '#00d4ff'
  context.fillText(hash.slice(0, 10), 256, 280)
  context.fillText(hash.slice(10, 20), 256, 320)
  
  const texture = new THREE.CanvasTexture(canvas)
  
  // 将文字纹理应用到正面
  const textMaterial = new THREE.MeshBasicMaterial({ 
    map: texture, 
    transparent: true,
    opacity: 0.9
  })
  const textGeometry = new THREE.PlaneGeometry(size * 0.95, size * 0.95)
  const textMesh = new THREE.Mesh(textGeometry, textMaterial)
  textMesh.position.z = size / 2 + 0.01
  group.add(textMesh)
  
  // 新区块特效
  if (isNew) {
    // 发光球
    const glowGeometry = new THREE.SphereGeometry(size * 0.8, 32, 32)
    const glowMaterial = new THREE.MeshBasicMaterial({
      color: 0x00d4ff,
      transparent: true,
      opacity: 0.4,
      blending: THREE.AdditiveBlending
    })
    const glow = new THREE.Mesh(glowGeometry, glowMaterial)
    group.add(glow)
    
    // 1.5秒后移除发光效果
    setTimeout(() => {
      group.remove(glow)
      material.emissiveIntensity = 0.15
    }, 1500)
  }
  
  return group
}

function addBlock(
  height: number, 
  hash: string, 
  isNew: boolean, 
  timestamp?: number,
  blockInfo?: BlockInfo
) {
  const blockMesh = createBlockMesh(height, hash, isNew)
  // 使用initialOffset使第一个区块从左侧开始
  blockMesh.position.x = initialOffset + nextPosition * blockSpacing
  blockMesh.position.y = 0
  blockMesh.position.z = 0
  
  // 存储区块信息到mesh的userData中，方便点击时获取
  blockMesh.userData = {
    height,
    hash,
    timestamp: timestamp || Date.now(),
    txCount: blockInfo?.txCount || Math.floor(Math.random() * 200) + 10,
    size: blockInfo?.size || Math.floor(Math.random() * 500000) + 50000,
    miner: blockInfo?.miner || ('0x' + Array.from({ length: 40 }, () => Math.floor(Math.random() * 16).toString(16)).join(''))
  }
  
  if (scene) {
    scene.add(blockMesh)
  }
  
  // 创建箭头连接前一个区块
  let arrow: THREE.Group | undefined
  if (blocks.length > 0) {
    arrow = createArrow()
    arrow.position.x = initialOffset + (nextPosition - 0.5) * blockSpacing
    arrow.position.y = 0
    arrow.position.z = 0
    
    if (scene) {
      scene.add(arrow)
    }
  }
  
  blocks.push({
    mesh: blockMesh,
    arrow,
    height,
    hash,
    position: nextPosition,
    timestamp: blockMesh.userData.timestamp,
    txCount: blockMesh.userData.txCount,
    size: blockMesh.userData.size,
    miner: blockMesh.userData.miner
  })
  
  nextPosition++
  
  // 移除过多的区块和箭头
  if (blocks.length > maxBlocks) {
    const removed = blocks.shift()
    if (removed && scene) {
      scene.remove(removed.mesh)
      if (removed.arrow) {
        scene.remove(removed.arrow)
      }
    }
  }
  
  // 移动场景跟随最新区块(而不是移动相机)
  // 只有当区块链条接近右侧边缘时才开始向左移动
  if (isNew && scene && camera) {
    // 安全地计算可见区块数
    const viewSize = 4
    const aspect = camera.right / viewSize
    
    // 检查aspect是否有效
    if (isFinite(aspect) && aspect > 0) {
      const visibleWidth = 2 * viewSize * aspect
      // 使用blockSpacing计算可见区块数
      const visibleBlocks = Math.floor(visibleWidth / blockSpacing)
      
      // 当区块数量超过可见范围时,开始滚动
      // 留出2个区块的边距,当最新区块到达右侧边缘附近时开始移动
      const scrollThreshold = Math.max(6, visibleBlocks - 2)
      
      if (nextPosition > scrollThreshold) {
        // 计算场景偏移,保持最新区块在右侧可见
        const targetOffset = -(nextPosition - scrollThreshold) * blockSpacing
        animateSceneTo(targetOffset)
      }
    }
  }
}

function createArrow(): THREE.Group {
  const group = new THREE.Group()
  
  const arrowLength = blockSpacing - 1.2  // 根据新的blockSpacing调整
  
  // 箭头杆 - 从左指向右
  const shaftGeometry = new THREE.CylinderGeometry(0.05, 0.05, arrowLength, 8)
  const shaftMaterial = new THREE.MeshStandardMaterial({ 
    color: 0xffd700,
    emissive: 0xffaa00,
    emissiveIntensity: 0.4,
    metalness: 0.8,
    roughness: 0.2
  })
  const shaft = new THREE.Mesh(shaftGeometry, shaftMaterial)
  shaft.rotation.z = Math.PI / 2
  group.add(shaft)
  
  // 箭头头部（圆锥）
  const headGeometry = new THREE.ConeGeometry(0.15, 0.3, 8)
  const head = new THREE.Mesh(headGeometry, shaftMaterial)
  head.rotation.z = -Math.PI / 2
  head.position.x = arrowLength / 2 + 0.15
  group.add(head)
  
  // 添加发光效果
  const glowGeometry = new THREE.CylinderGeometry(0.08, 0.08, arrowLength, 8)
  const glowMaterial = new THREE.MeshBasicMaterial({
    color: 0xffd700,
    transparent: true,
    opacity: 0.3,
    blending: THREE.AdditiveBlending
  })
  const glow = new THREE.Mesh(glowGeometry, glowMaterial)
  glow.rotation.z = Math.PI / 2
  group.add(glow)
  
  return group
}

// 不再需要 addNewBlock 和 generateHash 函数，由后端提供真实数据

function animateSceneTo(targetX: number) {
  if (!scene) return
  
  const startX = scene.position.x
  const duration = 1000
  const startTime = Date.now()
  
  function update() {
    const elapsed = Date.now() - startTime
    const progress = Math.min(elapsed / duration, 1)
    const eased = easeInOutCubic(progress)
    
    if (scene) {
      // 移动整个场景，相机保持不动
      scene.position.x = startX + (targetX - startX) * eased
    }
    
    if (progress < 1) {
      requestAnimationFrame(update)
    }
  }
  
  update()
}

function easeInOutCubic(t: number): number {
  return t < 0.5 ? 4 * t * t * t : 1 - Math.pow(-2 * t + 2, 3) / 2
}

function animate() {
  animationId = requestAnimationFrame(animate)
  
  const time = Date.now() * 0.001
  
  // 确保相机始终看向原点，保持视角稳定
  if (camera) {
    camera.up.set(0, 1, 0)  // 确保每帧up向量正确
    camera.lookAt(0, 0, 0)
  }
  
  // 箭头脉动效果
  blocks.forEach((block, index) => {
    if (block.arrow) {
      const scale = 1 + Math.sin(time * 3 + index) * 0.08
      block.arrow.scale.set(scale, 1, 1)
    }
  })
  
  if (renderer && scene && camera) {
    renderer.render(scene, camera)
  }
}

function handleResize() {
  if (!containerRef.value || !camera || !renderer) return
  
  const width = containerRef.value.clientWidth
  const height = containerRef.value.clientHeight
  const aspect = width / height
  const viewSize = 4
  
  camera.left = -viewSize * aspect
  camera.right = viewSize * aspect
  camera.top = viewSize
  camera.bottom = -viewSize
  camera.updateProjectionMatrix()
  renderer.setSize(width, height)
  
  // 窗口大小变化时重新计算最大块数
  const visibleWidth = 2 * viewSize * aspect
  const newMaxBlocks = Math.max(6, Math.floor(visibleWidth / blockSpacing))
  
  // 如果最大块数变化，可能需要移除或保留更多区块
  if (newMaxBlocks !== maxBlocks) {
    console.log(`窗口大小变化，最大块数从 ${maxBlocks} 更新为 ${newMaxBlocks}`)
    maxBlocks = newMaxBlocks
    
    // 如果新的最大块数更小，移除多余的旧区块
    while (blocks.length > maxBlocks) {
      const removed = blocks.shift()
      if (removed && scene) {
        scene.remove(removed.mesh)
        if (removed.arrow) {
          scene.remove(removed.arrow)
        }
      }
    }
  }
}

function onBlockClick(event: MouseEvent) {
  if (!containerRef.value || !camera || !raycaster || !mouse || !scene) return
  
  const rect = containerRef.value.getBoundingClientRect()
  mouse.x = ((event.clientX - rect.left) / rect.width) * 2 - 1
  mouse.y = -((event.clientY - rect.top) / rect.height) * 2 + 1
  
  raycaster.setFromCamera(mouse, camera)
  
  // 只检测区块mesh（不包括箭头）
  const blockMeshes = blocks.map(b => b.mesh)
  const intersects = raycaster.intersectObjects(blockMeshes, true)
  
  if (intersects.length > 0) {
    // 找到点击的区块
    let clickedMesh = intersects[0].object
    while (clickedMesh.parent && clickedMesh.parent.type !== 'Scene') {
      clickedMesh = clickedMesh.parent
    }
    
    const blockData = clickedMesh.userData
    if (blockData && blockData.height) {
      // 从后端获取详细信息
      fetchBlockDetail(blockData.height)
    }
  }
}

// 获取区块详情
async function fetchBlockDetail(height: number) {
  try {
    const detail = await api.getBlockByHeight(height)
    selectedBlock.value = {
      height: detail.height,
      hash: detail.hash,
      timestamp: detail.timestamp * 1000, // 转换为毫秒
      txCount: detail.txCount,
      size: detail.size,
      miner: detail.miner
    }
    showBlockDetail.value = true
  } catch (error) {
    console.error(`Failed to fetch block detail for height ${height}:`, error)
    // 如果获取失败，使用本地数据
    const block = blocks.find(b => b.height === height)
    if (block) {
      selectedBlock.value = {
        height: block.height,
        hash: block.hash,
        timestamp: block.timestamp,
        txCount: block.txCount,
        size: block.size,
        miner: block.miner
      }
      showBlockDetail.value = true
    }
  }
}

function onMouseMove(event: MouseEvent) {
  if (!containerRef.value || !camera || !raycaster || !mouse || !renderer) return
  
  const rect = containerRef.value.getBoundingClientRect()
  mouse.x = ((event.clientX - rect.left) / rect.width) * 2 - 1
  mouse.y = -((event.clientY - rect.top) / rect.height) * 2 + 1
  
  raycaster.setFromCamera(mouse, camera)
  
  const blockMeshes = blocks.map(b => b.mesh)
  const intersects = raycaster.intersectObjects(blockMeshes, true)
  
  // 改变鼠标样式
  if (renderer && renderer.domElement) {
    renderer.domElement.style.cursor = intersects.length > 0 ? 'pointer' : 'default'
  }
}

function cleanup() {
  if (animationId !== null) {
    cancelAnimationFrame(animationId)
  }
  
  if (renderer && containerRef.value) {
    renderer.domElement.removeEventListener('click', onBlockClick)
    renderer.domElement.removeEventListener('mousemove', onMouseMove)
    containerRef.value.removeChild(renderer.domElement)
    renderer.dispose()
  }
  
  window.removeEventListener('resize', handleResize)
}

// 导出组件（修复 no default export 错误）
defineExpose({
  cleanup
})
</script>

<style scoped>
.blockchain-3d-wrapper {
  width: 100%;
  height: 100%;
  position: relative;
}

.blockchain-3d-container {
  width: 100%;
  height: 100%;
  position: relative;
  background: transparent;
  border-radius: 8px;
  overflow: hidden;
}

.blockchain-3d-container canvas {
  display: block;
}
</style>
