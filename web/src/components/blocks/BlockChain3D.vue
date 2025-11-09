<template>
  <div class="blockchain-3d-container" ref="containerRef"></div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import * as THREE from 'three'
import { mockService } from '@/services/mock'

const containerRef = ref<HTMLElement | null>(null)

let scene: THREE.Scene | null = null
let camera: THREE.PerspectiveCamera | null = null
let renderer: THREE.WebGLRenderer | null = null
let animationId: number | null = null

interface Block {
  mesh: THREE.Group
  height: number
  hash: string
  position: number
}

const blocks: Block[] = []
const blockSpacing = 2.8  // 调整间距适配更大的立方体
const maxBlocks = 20
let nextPosition = 0

onMounted(() => {
  if (!containerRef.value) return
  
  initScene()
  initBlocks()
  animate()
  
  // 定期添加新区块
  const interval = setInterval(() => {
    addNewBlock()
  }, 3000)
  
  onUnmounted(() => {
    clearInterval(interval)
    cleanup()
  })
  
  window.addEventListener('resize', handleResize)
})

function initScene() {
  if (!containerRef.value) return
  
  const width = containerRef.value.clientWidth
  const height = containerRef.value.clientHeight
  
  // 场景
  scene = new THREE.Scene()
  scene.fog = new THREE.Fog(0x0a0e27, 10, 50)
  
  // 相机 - 优化视角，减少上下空白，增大FOV使区块更充满画面
  camera = new THREE.PerspectiveCamera(45, width / height, 0.1, 1000)
  camera.position.set(0, 0.8, 5.5)
  camera.lookAt(0, 0, 0)
  
  // 渲染器
  renderer = new THREE.WebGLRenderer({ 
    alpha: true, 
    antialias: true 
  })
  renderer.setSize(width, height)
  renderer.setPixelRatio(window.devicePixelRatio)
  renderer.setClearColor(0x0a0e27, 0.3)
  containerRef.value.appendChild(renderer.domElement)
  
  // 光源
  const ambientLight = new THREE.AmbientLight(0xffffff, 0.5)
  scene.add(ambientLight)
  
  const directionalLight = new THREE.DirectionalLight(0xffffff, 0.8)
  directionalLight.position.set(5, 10, 7)
  scene.add(directionalLight)
  
  const pointLight = new THREE.PointLight(0x00d4ff, 1, 50)
  pointLight.position.set(-10, 5, 0)
  scene.add(pointLight)
}

function initBlocks() {
  // 初始化一些区块
  for (let i = 0; i < 8; i++) {
    addBlock(i + 12000, generateHash(), false)
  }
}

function createBlockMesh(height: number, hash: string, isNew: boolean): THREE.Group {
  const group = new THREE.Group()
  
  // 立方体几何体 - 稍微增大尺寸
  const geometry = new THREE.BoxGeometry(1.8, 1.8, 1.8)
  
  // 材质 - 带发光效果
  const material = new THREE.MeshPhongMaterial({
    color: 0x00d4ff,
    emissive: isNew ? 0x00d4ff : 0x004466,
    emissiveIntensity: isNew ? 0.5 : 0.2,
    shininess: 100,
    transparent: true,
    opacity: 0.9
  })
  
  const cube = new THREE.Mesh(geometry, material)
  group.add(cube)
  
  // 边框
  const edges = new THREE.EdgesGeometry(geometry)
  const lineMaterial = new THREE.LineBasicMaterial({ 
    color: 0xffffff, 
    transparent: true, 
    opacity: 0.6 
  })
  const wireframe = new THREE.LineSegments(edges, lineMaterial)
  group.add(wireframe)
  
  // 创建文字纹理
  const canvas = document.createElement('canvas')
  const context = canvas.getContext('2d')!
  canvas.width = 256
  canvas.height = 256
  
  context.fillStyle = '#ffffff'
  context.font = 'bold 32px Arial'
  context.textAlign = 'center'
  context.textBaseline = 'middle'
  context.fillText(`#${height}`, 128, 80)
  
  context.font = '16px monospace'
  context.fillText(hash.slice(0, 8) + '...', 128, 140)
  
  const texture = new THREE.CanvasTexture(canvas)
  
  // 将文字纹理应用到前面 - 调整尺寸和位置
  const textMaterial = new THREE.MeshBasicMaterial({ 
    map: texture, 
    transparent: true 
  })
  const textGeometry = new THREE.PlaneGeometry(1.7, 1.7)
  const textMesh = new THREE.Mesh(textGeometry, textMaterial)
  textMesh.position.z = 0.91
  group.add(textMesh)
  
  // 新区块发光效果
  if (isNew) {
    const glowGeometry = new THREE.SphereGeometry(1.5, 16, 16)
    const glowMaterial = new THREE.MeshBasicMaterial({
      color: 0x00d4ff,
      transparent: true,
      opacity: 0.3
    })
    const glow = new THREE.Mesh(glowGeometry, glowMaterial)
    group.add(glow)
    
    // 1秒后移除发光效果
    setTimeout(() => {
      group.remove(glow)
      material.emissiveIntensity = 0.2
    }, 1000)
  }
  
  return group
}

function addBlock(height: number, hash: string, isNew: boolean) {
  const blockMesh = createBlockMesh(height, hash, isNew)
  blockMesh.position.x = nextPosition * blockSpacing
  
  if (scene) {
    scene.add(blockMesh)
  }
  
  blocks.push({
    mesh: blockMesh,
    height,
    hash,
    position: nextPosition
  })
  
  nextPosition++
  
  // 创建箭头指向下一个区块
  if (blocks.length > 1) {
    const arrow = createArrow()
    arrow.position.x = (nextPosition - 1) * blockSpacing - blockSpacing / 2
    arrow.position.y = 0
    if (scene) {
      scene.add(arrow)
    }
  }
  
  // 移除过多的区块
  if (blocks.length > maxBlocks) {
    const removed = blocks.shift()
    if (removed && scene) {
      scene.remove(removed.mesh)
    }
  }
  
  // 平滑移动相机
  if (isNew && camera) {
    const targetX = (nextPosition - 5) * blockSpacing
    animateCameraTo(targetX)
  }
}

function createArrow(): THREE.Group {
  const group = new THREE.Group()
  
  // 箭头杆
  const shaftGeometry = new THREE.CylinderGeometry(0.05, 0.05, 1, 8)
  const shaftMaterial = new THREE.MeshPhongMaterial({ 
    color: 0xffd700,
    emissive: 0x886600,
    emissiveIntensity: 0.3
  })
  const shaft = new THREE.Mesh(shaftGeometry, shaftMaterial)
  shaft.rotation.z = Math.PI / 2
  group.add(shaft)
  
  // 箭头头部
  const headGeometry = new THREE.ConeGeometry(0.15, 0.3, 8)
  const head = new THREE.Mesh(headGeometry, shaftMaterial)
  head.rotation.z = -Math.PI / 2
  head.position.x = 0.65
  group.add(head)
  
  return group
}

function addNewBlock() {
  const lastBlock = blocks[blocks.length - 1]
  const newHeight = lastBlock ? lastBlock.height + 1 : 12000
  const newHash = generateHash()
  
  addBlock(newHeight, newHash, true)
  mockService.incrementBlock()
}

function generateHash(): string {
  return '0x' + Array.from({ length: 12 }, () => 
    Math.floor(Math.random() * 16).toString(16)
  ).join('')
}

function animateCameraTo(targetX: number) {
  if (!camera) return
  
  const startX = camera.position.x
  const duration = 800
  const startTime = Date.now()
  
  function update() {
    const elapsed = Date.now() - startTime
    const progress = Math.min(elapsed / duration, 1)
    const eased = easeInOutCubic(progress)
    
    if (camera) {
      camera.position.x = startX + (targetX - startX) * eased
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
  
  // 旋转所有区块
  blocks.forEach(block => {
    block.mesh.rotation.y += 0.005
  })
  
  if (renderer && scene && camera) {
    renderer.render(scene, camera)
  }
}

function handleResize() {
  if (!containerRef.value || !camera || !renderer) return
  
  const width = containerRef.value.clientWidth
  const height = containerRef.value.clientHeight
  
  camera.aspect = width / height
  camera.updateProjectionMatrix()
  renderer.setSize(width, height)
}

function cleanup() {
  if (animationId !== null) {
    cancelAnimationFrame(animationId)
  }
  
  if (renderer && containerRef.value) {
    containerRef.value.removeChild(renderer.domElement)
    renderer.dispose()
  }
  
  window.removeEventListener('resize', handleResize)
}
</script>

<style scoped>
.blockchain-3d-container {
  width: 100%;
  height: 100%;
  position: relative;
  background: linear-gradient(135deg, #0a0e27 0%, #141d3a 100%);
  border-radius: 8px;
  overflow: hidden;
}

.blockchain-3d-container canvas {
  display: block;
}
</style>
