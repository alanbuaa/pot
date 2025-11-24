<template>
  <div class="reward-pie-chart" ref="containerRef"></div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, watch } from 'vue'
import * as THREE from 'three'

interface Props {
  miner: number
  exchequer: number
  committeeLeader: number
  committeeMember: number
}

const props = defineProps<Props>()
const containerRef = ref<HTMLElement | null>(null)

let scene: THREE.Scene | null = null
let camera: THREE.PerspectiveCamera | null = null
let renderer: THREE.WebGLRenderer | null = null
let animationId: number | null = null
let slices: THREE.Group[] = []

onMounted(() => {
  if (!containerRef.value) return
  
  initScene()
  createPieChart()
  animate()
  
  window.addEventListener('resize', handleResize)
})

onUnmounted(() => {
  cleanup()
  window.removeEventListener('resize', handleResize)
})

watch(() => [props.miner, props.exchequer, props.committeeLeader, props.committeeMember], () => {
  updatePieChart()
})

function initScene() {
  if (!containerRef.value) return
  
  const width = containerRef.value.clientWidth
  const height = containerRef.value.clientHeight
  
  // 场景
  scene = new THREE.Scene()
  
  // 相机
  camera = new THREE.PerspectiveCamera(45, width / height, 0.1, 1000)
  camera.position.set(0, 4, 5)
  camera.lookAt(0, 0, 0)
  
  // 渲染器
  renderer = new THREE.WebGLRenderer({ 
    alpha: true, 
    antialias: true 
  })
  renderer.setSize(width, height)
  renderer.setPixelRatio(Math.min(window.devicePixelRatio, 2))
  renderer.setClearColor(0x000000, 0)
  containerRef.value.appendChild(renderer.domElement)
  
  // 光源
  const ambientLight = new THREE.AmbientLight(0xffffff, 0.6)
  scene.add(ambientLight)
  
  const directionalLight = new THREE.DirectionalLight(0xffffff, 0.8)
  directionalLight.position.set(5, 5, 5)
  scene.add(directionalLight)
  
  const pointLight = new THREE.PointLight(0xffffff, 0.5)
  pointLight.position.set(-3, 3, 3)
  scene.add(pointLight)
}

function createPieChart() {
  updatePieChart()
}

function updatePieChart() {
  if (!scene) return
  
  // 清除旧的切片
  slices.forEach(slice => scene?.remove(slice))
  slices = []
  
  const total = props.miner + props.exchequer + props.committeeLeader + props.committeeMember
  if (total === 0) return
  
  const radius = 1.5
  const depth = 0.4
  const data = [
    { value: props.miner, color: 0x4a9eff, label: '矿工' },
    { value: props.exchequer, color: 0xff6b6b, label: '国库' },
    { value: props.committeeLeader, color: 0xffd700, label: '委员会Leader' },
    { value: props.committeeMember, color: 0x51cf66, label: '委员会成员' }
  ]
  
  let startAngle = -Math.PI / 2 // 从顶部开始
  
  data.forEach((item) => {
    if (item.value === 0) return
    
    const angle = (item.value / total) * Math.PI * 2
    const slice = createSlice(radius, depth, startAngle, angle, item.color)
    
    if (scene) {
      scene.add(slice)
      slices.push(slice)
    }
    
    startAngle += angle
  })
}

function createSlice(
  radius: number, 
  depth: number, 
  startAngle: number, 
  angle: number, 
  color: number
): THREE.Group {
  const group = new THREE.Group()
  
  // 创建扇形形状
  const shape = new THREE.Shape()
  shape.moveTo(0, 0)
  
  const segments = Math.max(3, Math.ceil(angle / (Math.PI / 16)))
  for (let i = 0; i <= segments; i++) {
    const a = startAngle + (angle * i) / segments
    const x = Math.cos(a) * radius
    const y = Math.sin(a) * radius
    shape.lineTo(x, y)
  }
  shape.lineTo(0, 0)
  
  // 挤出几何体
  const extrudeSettings = {
    depth: depth,
    bevelEnabled: true,
    bevelThickness: 0.05,
    bevelSize: 0.05,
    bevelSegments: 3
  }
  
  const geometry = new THREE.ExtrudeGeometry(shape, extrudeSettings)
  
  // 材质
  const material = new THREE.MeshStandardMaterial({
    color: color,
    emissive: color,
    emissiveIntensity: 0.2,
    metalness: 0.3,
    roughness: 0.4
  })
  
  const mesh = new THREE.Mesh(geometry, material)
  mesh.position.z = -depth / 2
  group.add(mesh)
  
  // 边框线
  const edges = new THREE.EdgesGeometry(geometry)
  const lineMaterial = new THREE.LineBasicMaterial({ 
    color: 0xffffff, 
    transparent: true, 
    opacity: 0.3 
  })
  const wireframe = new THREE.LineSegments(edges, lineMaterial)
  wireframe.position.z = -depth / 2
  group.add(wireframe)
  
  // 轻微偏移每个切片以产生分离效果
  const midAngle = startAngle + angle / 2
  const offsetDistance = 0.05
  group.position.x = Math.cos(midAngle) * offsetDistance
  group.position.y = Math.sin(midAngle) * offsetDistance
  
  // 沿Y轴倾斜使饼图呈扁圆状
  group.rotation.x = Math.PI / 10 +  Math.PI
  
  return group
}

function animate() {
  animationId = requestAnimationFrame(animate)
  
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
  
  slices.forEach(slice => {
    slice.traverse((obj) => {
      if (obj instanceof THREE.Mesh) {
        obj.geometry.dispose()
        if (Array.isArray(obj.material)) {
          obj.material.forEach(mat => mat.dispose())
        } else {
          obj.material.dispose()
        }
      }
    })
  })
  slices = []
}
</script>

<style scoped>
.reward-pie-chart {
  width: 100%;
  height: 120px;
  position: relative;
}

.reward-pie-chart canvas {
  display: block;
}
</style>
