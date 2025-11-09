<template>
  <div ref="containerRef" class="three-background"></div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import * as THREE from 'three'

const containerRef = ref<HTMLElement | null>(null)
let scene: THREE.Scene | null = null
let camera: THREE.PerspectiveCamera | null = null
let renderer: THREE.WebGLRenderer | null = null
let particles: THREE.Points | null = null
let animationId: number | null = null

onMounted(() => {
  if (!containerRef.value) return
  
  // 初始化场景
  scene = new THREE.Scene()
  
  // 初始化相机
  camera = new THREE.PerspectiveCamera(
    75,
    window.innerWidth / window.innerHeight,
    0.1,
    1000
  )
  camera.position.z = 50
  
  // 初始化渲染器
  renderer = new THREE.WebGLRenderer({ 
    alpha: true,
    antialias: true 
  })
  renderer.setSize(window.innerWidth, window.innerHeight)
  renderer.setPixelRatio(window.devicePixelRatio)
  containerRef.value.appendChild(renderer.domElement)
  
  // 创建粒子系统
  createParticles()
  
  // 添加光效
  createLights()
  
  // 开始动画循环
  animate()
  
  // 监听窗口大小变化
  window.addEventListener('resize', handleResize)
})

onUnmounted(() => {
  window.removeEventListener('resize', handleResize)
  
  if (animationId !== null) {
    cancelAnimationFrame(animationId)
  }
  
  if (renderer && containerRef.value) {
    containerRef.value.removeChild(renderer.domElement)
    renderer.dispose()
  }
})

function createParticles() {
  if (!scene) return
  
  const particleCount = 2000
  const geometry = new THREE.BufferGeometry()
  const positions = new Float32Array(particleCount * 3)
  const colors = new Float32Array(particleCount * 3)
  
  // POT 主题色
  const color1 = new THREE.Color(0x00d4ff) // 挖矿蓝
  const color2 = new THREE.Color(0xffd700) // 领导金
  const color3 = new THREE.Color(0x9370db) // 存储紫
  
  for (let i = 0; i < particleCount; i++) {
    const i3 = i * 3
    
    // 随机位置
    positions[i3] = (Math.random() - 0.5) * 200
    positions[i3 + 1] = (Math.random() - 0.5) * 200
    positions[i3 + 2] = (Math.random() - 0.5) * 200
    
    // 随机颜色（从主题色中选择）
    const colorChoice = Math.random()
    let color: THREE.Color
    if (colorChoice < 0.33) {
      color = color1
    } else if (colorChoice < 0.66) {
      color = color2
    } else {
      color = color3
    }
    
    colors[i3] = color.r
    colors[i3 + 1] = color.g
    colors[i3 + 2] = color.b
  }
  
  geometry.setAttribute('position', new THREE.BufferAttribute(positions, 3))
  geometry.setAttribute('color', new THREE.BufferAttribute(colors, 3))
  
  const material = new THREE.PointsMaterial({
    size: 0.5,
    vertexColors: true,
    transparent: true,
    opacity: 0.6,
    blending: THREE.AdditiveBlending
  })
  
  particles = new THREE.Points(geometry, material)
  scene.add(particles)
}

function createLights() {
  if (!scene) return
  
  // 环境光
  const ambientLight = new THREE.AmbientLight(0x404040, 0.5)
  scene.add(ambientLight)
  
  // 点光源 - 蓝色
  const pointLight1 = new THREE.PointLight(0x00d4ff, 1, 100)
  pointLight1.position.set(50, 50, 50)
  scene.add(pointLight1)
  
  // 点光源 - 金色
  const pointLight2 = new THREE.PointLight(0xffd700, 0.8, 100)
  pointLight2.position.set(-50, -50, 50)
  scene.add(pointLight2)
}

function animate() {
  animationId = requestAnimationFrame(animate)
  
  if (particles) {
    // 旋转粒子系统
    particles.rotation.x += 0.0002
    particles.rotation.y += 0.0003
    
    // 粒子脉动效果
    const positions = particles.geometry.attributes.position.array as Float32Array
    const time = Date.now() * 0.0005
    
    for (let i = 0; i < positions.length; i += 3) {
      positions[i + 1] += Math.sin(time + positions[i]) * 0.01
    }
    
    particles.geometry.attributes.position.needsUpdate = true
  }
  
  if (camera) {
    // 相机轻微摆动
    camera.position.x = Math.sin(Date.now() * 0.0001) * 5
    camera.position.y = Math.cos(Date.now() * 0.00015) * 5
    camera.lookAt(0, 0, 0)
  }
  
  if (renderer && scene && camera) {
    renderer.render(scene, camera)
  }
}

function handleResize() {
  if (!camera || !renderer) return
  
  camera.aspect = window.innerWidth / window.innerHeight
  camera.updateProjectionMatrix()
  renderer.setSize(window.innerWidth, window.innerHeight)
}
</script>

<style scoped>
.three-background {
  position: fixed;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  z-index: -1;
  pointer-events: none;
}

.three-background canvas {
  display: block;
}
</style>
