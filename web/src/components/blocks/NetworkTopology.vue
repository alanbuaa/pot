<template>
  <div class="network-topology h-full relative">
    <div class="flex items-center justify-between mb-4">
      <h3 class="text-sm font-semibold mb-3 flex items-center gap-2">
        节点网络
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
    <div
      ref="container"
      class="topology-container"
      style="height: calc(100% - 60px)"
    ></div>

    <!-- 图例 -->
    <div
      class="legend absolute bottom-4 right-4 bg-bg-primary bg-opacity-80 p-3 rounded border border-border"
    >
      <div class="text-xs space-y-2">
        <div class="flex items-center gap-2">
          <span class="w-3 h-3 rounded-full bg-blue-500"></span>
          <span>POT节点</span>
        </div>
        <div class="flex items-center gap-2">
          <span class="w-3 h-3 rounded-full bg-blue-400 animate-pulse"></span>
          <span>本机节点</span>
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
          <a-descriptions-item label="类型">
            <a-tag
              :color="
                getNodeTypeColor(
                  selectedNode.layer,
                  selectedNode.isLeader,
                  selectedNode.isLocal
                )
              "
            >
              {{
                getNodeTypeName(
                  selectedNode.layer,
                  selectedNode.isLeader,
                  selectedNode.isLocal
                )
              }}
            </a-tag>
          </a-descriptions-item>
          <a-descriptions-item label="层级">
            {{ selectedNode.layer === "committee" ? "委员会层" : "POT节点层" }}
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
import { ref, onMounted, onUnmounted, watch } from "vue";
import { storeToRefs } from "pinia";
import { FullscreenOutlined } from "@ant-design/icons-vue";
import { useNetworkStore } from "@/stores/network";
import * as THREE from "three";
import { OrbitControls } from "three/addons/controls/OrbitControls.js";
import mapImageUrl from "@/assets/images/map.png";

const container = ref<HTMLDivElement>();
const networkStore = useNetworkStore();
const { topology } = storeToRefs(networkStore);

const detailVisible = ref(false);
const selectedNode = ref<any>(null);

let scene: THREE.Scene;
let camera: THREE.PerspectiveCamera;
let renderer: THREE.WebGLRenderer;
let controls: OrbitControls;
let animationId: number;
let nodeObjects: Array<{ mesh: THREE.Mesh; data: any; pulsePhase: number; angle: number }> =
  [];
let connectionLines: Array<THREE.Line> = [];

// 节点配置
const NODE_COUNT = 40; // 总节点数
const EARTH_RADIUS = 14; // 地球半径（增大以更好填充视图）
const NODE_DISTANCE_FROM_CENTER = EARTH_RADIUS * 1.08; // 节点距离地球中心的距离
const NODE_SIZE = 0.18; // 节点大小
const EARTH_ROTATION_SPEED = 0.001; // 地球自转速度
const CONNECTION_THRESHOLD = EARTH_RADIUS * 0.8; // 节点连接距离阈值
const MESSAGE_SPEED = 0.015; // 信息流速度
const MESSAGE_INTERVAL = 2000; // 消息发送周期（毫秒）

let earthMesh: THREE.Mesh | null = null;
let networkGroup: THREE.Group | null = null; // 用于整体旋转节点网络
let messageParticles: Array<{
  particle: THREE.Line | THREE.Mesh;
  glowParticle?: THREE.Line; // 外层发光线段
  curve: THREE.QuadraticBezierCurve3;
  progress: number;
  connection: { from: number; to: number };
  color: number;
  level: number;
}> = [];

// 节点连接关系表（哪些节点相互连接）
let nodeConnections: Map<number, Set<number>> = new Map();

// 消息颜色池
const MESSAGE_COLORS = [
  0x00ffff, // 青色
  0xff00ff, // 紫红色
  0xffff00, // 黄色
  0x00ff00, // 绿色
  0xff6600, // 橙色
  0xff0099, // 粉红色
];
let currentColorIndex = 0;

function initThreeJS() {
  if (!container.value) return;

  const width = container.value.clientWidth;
  const height = container.value.clientHeight;

  // 创建场景
  scene = new THREE.Scene();
  scene.fog = new THREE.Fog(0x0a0e27, 10, 50);

  // 创建相机
  camera = new THREE.PerspectiveCamera(60, width / height, 0.1, 1000);
  camera.position.set(20, 16, 20); // 斜向视角，距离调整以适应更大的地球
  camera.lookAt(0, 0, 0);

  // 创建渲染器
  renderer = new THREE.WebGLRenderer({ antialias: true, alpha: true });
  renderer.setSize(width, height);
  renderer.setPixelRatio(window.devicePixelRatio);
  renderer.setClearColor(0x0a0e27, 0);
  container.value.appendChild(renderer.domElement);

  // 添加控制器
  controls = new OrbitControls(camera, renderer.domElement);
  controls.enableDamping = true;
  controls.dampingFactor = 0.05;
  controls.minDistance = 15;
  controls.maxDistance = 50;
  controls.maxPolarAngle = Math.PI / 2.2;

  // 添加光源
  const ambientLight = new THREE.AmbientLight(0xffffff, 0.6);
  scene.add(ambientLight);

  // 主光源（模拟太阳光）
  const directionalLight = new THREE.DirectionalLight(0xffffff, 1);
  directionalLight.position.set(5, 3, 5);
  scene.add(directionalLight);

  const pointLight1 = new THREE.PointLight(0x00d4ff, 0.8, 50);
  pointLight1.position.set(10, 10, 10);
  scene.add(pointLight1);

  const pointLight2 = new THREE.PointLight(0xff6b35, 0.6, 50);
  pointLight2.position.set(-10, 10, -10);
  scene.add(pointLight2);

  // 创建网络拓扑
  createNetworkTopology();

  // 添加点击事件
  renderer.domElement.addEventListener("click", onCanvasClick);

  // 窗口大小调整
  window.addEventListener("resize", onWindowResize);

  // 开始动画
  animate();
}

function createNetworkTopology() {
  nodeObjects = [];
  connectionLines = [];

  // 创建网络组用于整体旋转
  networkGroup = new THREE.Group();
  scene.add(networkGroup);

  // === 1. 创建中心 3D 地球 ===
  const textureLoader = new THREE.TextureLoader();
  
  // 创建地球球体
  const earthGeometry = new THREE.SphereGeometry(EARTH_RADIUS, 64, 64);
  
  // 加载地球纹理
  textureLoader.load(
    mapImageUrl,
    (texture) => {
      const earthMaterial = new THREE.MeshPhongMaterial({
        map: texture,
        bumpScale: 0.05,
        specular: new THREE.Color(0x333333),
        shininess: 5,
      });
      
      earthMesh = new THREE.Mesh(earthGeometry, earthMaterial);
      earthMesh.position.set(0, 0, 0);
      scene.add(earthMesh);
      
      // 创建节点和连接线（在纹理加载完成后）
      createNodesAndConnections();
      
      // 添加简单的大气光晕效果
      const glowGeometry = new THREE.SphereGeometry(EARTH_RADIUS * 1.05, 64, 64);
      const glowMaterial = new THREE.MeshBasicMaterial({
        color: 0x4488ff,
        transparent: true,
        opacity: 0.1,
        side: THREE.BackSide,
      });
      
      const glowMesh = new THREE.Mesh(glowGeometry, glowMaterial);
      scene.add(glowMesh);
    },
    undefined,
    (error) => {
      console.error('Error loading earth texture:', error);
      // 如果纹理加载失败，使用纯色球体
      const earthMaterial = new THREE.MeshPhongMaterial({
        color: 0x2233ff,
        emissive: 0x112244,
      });
      earthMesh = new THREE.Mesh(earthGeometry, earthMaterial);
      earthMesh.position.set(0, 0, 0);
      scene.add(earthMesh);
      
      // 创建节点和连接线
      createNodesAndConnections();
    }
  );
}

// 创建节点和连接线的函数
function createNodesAndConnections() {
  if (!earthMesh) return;

  // === 在球体表面随机分布 POT 节点（使用 Fibonacci 球面分布算法）===
  const goldenRatio = (1 + Math.sqrt(5)) / 2;
  const angleIncrement = Math.PI * 2 * goldenRatio;
  
  for (let i = 0; i < NODE_COUNT; i++) {
    // 使用 Fibonacci 球面分布获得均匀分布
    const t = i / NODE_COUNT;
    const inclination = Math.acos(1 - 2 * t);
    const azimuth = angleIncrement * i;
    
    const x = Math.sin(inclination) * Math.cos(azimuth) * NODE_DISTANCE_FROM_CENTER;
    const y = Math.sin(inclination) * Math.sin(azimuth) * NODE_DISTANCE_FROM_CENTER;
    const z = Math.cos(inclination) * NODE_DISTANCE_FROM_CENTER;
    
    const isLocal = i === 8; // 第9个节点是本机节点

    // 创建节点球体（更小）
    const geometry = new THREE.SphereGeometry(NODE_SIZE, 16, 16);
    const material = new THREE.MeshStandardMaterial({
      color: 0x5ba3ff, // 更亮的蓝色
      emissive: isLocal ? 0x5ba3ff : 0x3b82f6,
      emissiveIntensity: isLocal ? 1.2 : 0.6,
      metalness: 0.3,
      roughness: 0.2,
    });
    const nodeMesh = new THREE.Mesh(geometry, material);
    nodeMesh.position.set(x, y, z);
    earthMesh!.add(nodeMesh); // 添加到地球上，随地球一起旋转

    // 添加光晕效果（本机节点）
    if (isLocal) {
      const glowGeometry = new THREE.SphereGeometry(NODE_SIZE * 1.8, 16, 16);
      const glowMaterial = new THREE.MeshBasicMaterial({
        color: 0x5ba3ff,
        transparent: true,
        opacity: 0.6,
      });
      const glow = new THREE.Mesh(glowGeometry, glowMaterial);
      nodeMesh.add(glow);
    }

    nodeObjects.push({
      mesh: nodeMesh,
      data: {
        id: `pot-${i}`,
        layer: "pot",
        index: i,
        isLocal,
        isLeader: false,
      },
      pulsePhase: isLocal ? Math.random() * Math.PI * 2 : 0,
      angle: 0,
    });
  }

  // === 创建节点间的弧线连接（连接距离较近的节点）===
  for (let i = 0; i < NODE_COUNT; i++) {
    if (!nodeConnections.has(i)) {
      nodeConnections.set(i, new Set());
    }
    
    for (let j = i + 1; j < NODE_COUNT; j++) {
      const node1 = nodeObjects[i].mesh;
      const node2 = nodeObjects[j].mesh;
      
      // 计算两个节点之间的距离
      const distance = node1.position.distanceTo(node2.position);
      
      // 只连接距离较近的节点，形成网状结构
      if (distance < CONNECTION_THRESHOLD) {
        const arcLine = createArcLine(
          node1.position.clone(),
          node2.position.clone(),
          0x3b82f6,
          0.01
        );
        connectionLines.push(arcLine);
        earthMesh!.add(arcLine); // 添加到地球上，随地球一起旋转
        
        // 记录连接关系
        nodeConnections.get(i)!.add(j);
        if (!nodeConnections.has(j)) {
          nodeConnections.set(j, new Set());
        }
        nodeConnections.get(j)!.add(i);
      }
    }
  }
  
  // 启动周期性创建多级传播的信息流
  startMessageBroadcast();
}

// 启动多级消息传播
function startMessageBroadcast() {
  setInterval(() => {
    // 切换颜色
    currentColorIndex = (currentColorIndex + 1) % MESSAGE_COLORS.length;
    const messageColor = MESSAGE_COLORS[currentColorIndex];
    
    // 随机选择 1/3 的节点作为初始发送者
    const initiatorCount = Math.ceil(NODE_COUNT / 3);
    const initiators = new Set<number>();
    
    while (initiators.size < initiatorCount) {
      const randomNode = Math.floor(Math.random() * NODE_COUNT);
      initiators.add(randomNode);
    }
    
    // 记录哪些节点已经收到消息（避免重复传播）
    const receivedNodes = new Set<number>(initiators);
    
    // 第一级：初始节点发送消息给所有连接的节点
    const level1Targets = new Set<number>();
    initiators.forEach(fromNode => {
      const connections = nodeConnections.get(fromNode);
      if (connections) {
        connections.forEach(toNode => {
          if (!receivedNodes.has(toNode)) {
            level1Targets.add(toNode);
            receivedNodes.add(toNode);
            
            // 创建消息粒子
            const delay = Math.random() * 300;
            setTimeout(() => {
              createMessageParticleWithColor(fromNode, toNode, messageColor, 1);
            }, delay);
          }
        });
      }
    });
    
    // 第二级：第一级接收者再向外传播
    setTimeout(() => {
      level1Targets.forEach(fromNode => {
        const connections = nodeConnections.get(fromNode);
        if (connections) {
          connections.forEach(toNode => {
            if (!receivedNodes.has(toNode)) {
              receivedNodes.add(toNode);
              
              // 创建消息粒子
              const delay = Math.random() * 300;
              setTimeout(() => {
                createMessageParticleWithColor(fromNode, toNode, messageColor, 2);
              }, delay);
            }
          });
        }
      });
    }, 800); // 第二级延迟 800ms
    
  }, MESSAGE_INTERVAL);
}
function createArcLine(
  start: THREE.Vector3,
  end: THREE.Vector3,
  color: number,
  lineWidth: number
) {
  // 计算中点和控制点
  const midPoint = new THREE.Vector3().addVectors(start, end).multiplyScalar(0.5);
  
  // 将中点向外延伸,增加弧度(延伸到节点距离的1.3倍)
  const controlPoint = midPoint.normalize().multiplyScalar(NODE_DISTANCE_FROM_CENTER * 1.05);
  
  // 使用二次贝塞尔曲线创建弧线
  const curve = new THREE.QuadraticBezierCurve3(start, controlPoint, end);
  const points = curve.getPoints(30); // 增加到30个点使弧线更平滑
  const geometry = new THREE.BufferGeometry().setFromPoints(points);
  
  const material = new THREE.LineBasicMaterial({
    color,
    linewidth: lineWidth,
    transparent: true,
    opacity: 0.5,
  });
  
  const line = new THREE.Line(geometry, material);
  return line;
}

// 创建带颜色的信息流粒子
function createMessageParticleWithColor(fromIndex: number, toIndex: number, color: number, level: number) {
  if (!earthMesh) return;
  
  const node1 = nodeObjects[fromIndex].mesh;
  const node2 = nodeObjects[toIndex].mesh;
  
  // 计算曲线
  const midPoint = new THREE.Vector3().addVectors(node1.position, node2.position).multiplyScalar(0.5);
  const controlPoint = midPoint.normalize().multiplyScalar(NODE_DISTANCE_FROM_CENTER);
  const curve = new THREE.QuadraticBezierCurve3(node1.position.clone(), controlPoint, node2.position.clone());
  
  // 创建发光的短线段作为信息流
  const lineGeometry = new THREE.BufferGeometry();
  const positions = new Float32Array(6); // 两个点，每个点3个坐标
  lineGeometry.setAttribute('position', new THREE.BufferAttribute(positions, 3));
  
  const lineMaterial = new THREE.LineBasicMaterial({
    color: color,
    linewidth: 8, // 增加线宽
    transparent: true,
    opacity: 1.0,
  });
  
  const line = new THREE.Line(lineGeometry, lineMaterial);
  
  // 添加外层发光效果
  const glowGeometry = new THREE.BufferGeometry();
  const glowPositions = new Float32Array(6);
  glowGeometry.setAttribute('position', new THREE.BufferAttribute(glowPositions, 3));
  
  const glowMaterial = new THREE.LineBasicMaterial({
    color: color,
    linewidth: 16, // 更粗的外层光晕
    transparent: true,
    opacity: 0.4,
  });
  
  const glowLine = new THREE.Line(glowGeometry, glowMaterial);
  earthMesh.add(glowLine);
  earthMesh.add(line);
  
  messageParticles.push({
    particle: line as any, // 使用线段代替球体
    glowParticle: glowLine, // 外层发光线段
    curve,
    progress: 0,
    connection: { from: fromIndex, to: toIndex },
    color: color,
    level: level,
  });
}



function animate() {
  animationId = requestAnimationFrame(animate);

  // 地球自转（节点和连接线会随地球一起旋转）
  if (earthMesh) {
    earthMesh.rotation.y += EARTH_ROTATION_SPEED;
  }

  // 更新节点脉冲效果
  nodeObjects.forEach((obj) => {
    // 本机节点的脉冲效果
    if (obj.data.isLocal) {
      obj.pulsePhase += 0.05;
      const scale = 1 + Math.sin(obj.pulsePhase) * 0.2;
      obj.mesh.scale.set(scale, scale, scale);

      // 更新发光强度
      const material = obj.mesh.material as THREE.MeshStandardMaterial;
      material.emissiveIntensity = 0.8 + Math.sin(obj.pulsePhase) * 0.6;
    }
  });

  // 更新信息流粒子
  for (let i = messageParticles.length - 1; i >= 0; i--) {
    const mp = messageParticles[i];
    mp.progress += MESSAGE_SPEED;
    
    if (mp.progress >= 1) {
      // 粒子到达终点，移除
      earthMesh?.remove(mp.particle);
      if (mp.glowParticle) {
        earthMesh?.remove(mp.glowParticle);
      }
      messageParticles.splice(i, 1);
    } else {
      // 更新线段位置
      const segmentLength = 0.03; // 线段在曲线上占的比例
      const startProgress = Math.max(0, mp.progress - segmentLength);
      const endProgress = mp.progress;
      
      const startPoint = mp.curve.getPoint(startProgress);
      const endPoint = mp.curve.getPoint(endProgress);
      
      // 更新主线段
      const line = mp.particle as THREE.Line;
      const positions = line.geometry.attributes.position.array as Float32Array;
      positions[0] = startPoint.x;
      positions[1] = startPoint.y;
      positions[2] = startPoint.z;
      positions[3] = endPoint.x;
      positions[4] = endPoint.y;
      positions[5] = endPoint.z;
      line.geometry.attributes.position.needsUpdate = true;
      
      // 更新外层发光线段
      if (mp.glowParticle) {
        const glowPositions = mp.glowParticle.geometry.attributes.position.array as Float32Array;
        glowPositions[0] = startPoint.x;
        glowPositions[1] = startPoint.y;
        glowPositions[2] = startPoint.z;
        glowPositions[3] = endPoint.x;
        glowPositions[4] = endPoint.y;
        glowPositions[5] = endPoint.z;
        mp.glowParticle.geometry.attributes.position.needsUpdate = true;
        
        // 外层光晕透明度
        const glowMaterial = mp.glowParticle.material as THREE.LineBasicMaterial;
        glowMaterial.opacity = 0.4 * (1 - mp.progress * 0.2);
      }
      
      // 渐变透明度效果
      const material = line.material as THREE.LineBasicMaterial;
      material.opacity = 1.0 * (1 - mp.progress * 0.2);
    }
  }

  controls.update();
  renderer.render(scene, camera);
}

function onCanvasClick(event: MouseEvent) {
  const rect = container.value?.getBoundingClientRect();
  if (!rect) return;

  const mouse = new THREE.Vector2();
  mouse.x = ((event.clientX - rect.left) / rect.width) * 2 - 1;
  mouse.y = -((event.clientY - rect.top) / rect.height) * 2 + 1;

  const raycaster = new THREE.Raycaster();
  raycaster.setFromCamera(mouse, camera);

  const meshes = nodeObjects.map((obj) => obj.mesh);
  const intersects = raycaster.intersectObjects(meshes);

  if (intersects.length > 0) {
    const clickedMesh = intersects[0].object as THREE.Mesh;
    const nodeObj = nodeObjects.find((obj) => obj.mesh === clickedMesh);
    if (nodeObj) {
      selectedNode.value = nodeObj.data;
      detailVisible.value = true;
    }
  }
}

function onWindowResize() {
  if (!container.value) return;

  const width = container.value.clientWidth;
  const height = container.value.clientHeight;

  camera.aspect = width / height;
  camera.updateProjectionMatrix();
  renderer.setSize(width, height);
}

function resetCamera() {
  camera.position.set(20, 16, 20);
  camera.lookAt(0, 0, 0);
  controls.target.set(0, 0, 0);
  controls.update();
}

function getNodeTypeColor(layer: string, isLeader: boolean, isLocal: boolean) {
  if (isLocal) return "blue";
  if (layer === "committee") {
    return isLeader ? "orange" : "green";
  }
  return "blue";
}

function getNodeTypeName(layer: string, isLeader: boolean, isLocal: boolean) {
  if (isLocal) return "本机POT节点";
  if (layer === "committee") {
    return isLeader ? "委员会领导者" : "委员会成员";
  }
  return "POT节点";
}

watch(
  () => topology.value,
  () => {
    // 如果需要根据实时数据更新，可以在这里重新创建拓扑
  },
  { deep: true }
);

onMounted(() => {
  initThreeJS();
});

onUnmounted(() => {
  if (animationId) {
    cancelAnimationFrame(animationId);
  }
  if (renderer) {
    renderer.dispose();
  }
  if (container.value && renderer) {
    container.value.removeChild(renderer.domElement);
  }
  window.removeEventListener("resize", onWindowResize);
});
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
  0%,
  100% {
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
