<template>
  <div class="app min-h-screen text-white overflow-hidden">
    <!-- Three.js 3D 背景 -->
    <ThreeBackground />

    <!-- 顶部状态栏 -->
    <TopBar />

    <!-- 主内容区域 -->
    <div class="main-content pt-3 px-3" style="height: calc(100vh - 100px)">
      <div class="flex flex-col h-full gap-3">
        <!-- 上半部分：左中右三段对称 -->
        <div class="flex-1 grid grid-cols-10 gap-3 min-h-0">
          <!-- 左侧面板 (30%) -->
          <div class="col-span-3 flex flex-col gap-2 overflow-y-auto">
            <PotConsensus />
            <BCIIncentive />
          </div>

          <!-- 中心网络拓扑 (40%) -->
          <div class="col-span-4 flex flex-col gap-2">
            <div class="flex-1">
              <NetworkTopology />
            </div>
            <div class="flex-shrink-0">
              <CommitteeInfo />
            </div>
          </div>

          <!-- 右侧面板 (30%) -->
          <div class="col-span-3 flex flex-col gap-2 overflow-y-auto">
            <MempoolMonitor />
            <VDFMonitor />
          </div>
        </div>

        <!-- 下半部分：区块链可视化 -->
        <div class="h-16 flex-shrink-0 border-4 rounded-none" style="border-color: rgba(59, 130, 246, 0.5);">
          <BlockChain3D />
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, onUnmounted } from "vue";
import ThreeBackground from "@/components/ThreeBackground.vue";
import TopBar from "@/components/layout/TopBar.vue";
import PotConsensus from "@/components/blocks/PotConsensus.vue";
import VDFMonitor from "@/components/blocks/VDFMonitor.vue";
import PerformanceMetrics from "@/components/blocks/PerformanceMetrics.vue";
import NetworkTopology from "@/components/blocks/NetworkTopology.vue";
import MempoolMonitor from "@/components/blocks/MempoolMonitor.vue";
import BCIIncentive from "@/components/blocks/BCIIncentive.vue";
import CommitteeInfo from "@/components/blocks/CommitteeInfo.vue";
import SystemInfo from "@/components/blocks/SystemInfo.vue";
import BlockChain3D from "@/components/blocks/BlockChain3D.vue";

import { useSystemStore } from "@/stores/system";
import { usePotStore } from "@/stores/pot";
import { useCommitteeStore } from "@/stores/committee";
import { useNetworkStore } from "@/stores/network";
import { useMempoolStore } from "@/stores/mempool";
import { useBCIStore } from "@/stores/bci";
import { WebSocketService } from "@/services/websocket";

const systemStore = useSystemStore();
const potStore = usePotStore();
const committeeStore = useCommitteeStore();
const networkStore = useNetworkStore();
const mempoolStore = useMempoolStore();
const bciStore = useBCIStore();

let wsService: WebSocketService | null = null;

// 轮询定时器
let timers: number[] = [];

onMounted(async () => {
  // 初始数据加载
  await Promise.all([
    systemStore.fetchOverview(),
    potStore.fetchStatus(),
    potStore.fetchVDF(),
    committeeStore.fetchStatus(),
    mempoolStore.fetchStatus(),
    networkStore.fetchTopology(),
    bciStore.fetchStatus(),
  ]);

  // 设置轮询
  // 1秒轮询：系统概览、VDF
  timers.push(
    window.setInterval(() => {
      systemStore.fetchOverview();
      potStore.fetchVDF();
    }, 1000)
  );

  // 5秒轮询：POT状态、委员会、交易池
  timers.push(
    window.setInterval(() => {
      potStore.fetchStatus();
      committeeStore.fetchStatus();
      mempoolStore.fetchStatus();
    }, 5000)
  );

  // 10秒轮询：网络拓扑、BCI
  timers.push(
    window.setInterval(() => {
      networkStore.fetchTopology();
      bciStore.fetchStatus();
    }, 10000)
  );

  // 尝试连接 WebSocket
  try {
    const wsUrl = import.meta.env.VITE_WS_URL || "ws://localhost:8080/api/ws";
    wsService = new WebSocketService(wsUrl);
    wsService.connect();

    // 订阅实时更新
    setTimeout(() => {
      wsService?.subscribe([
        "pot",
        "vdf",
        "committee",
        "mempool",
        "network",
        "system",
      ]);
    }, 1000);

    // 监听 WebSocket 数据
    wsService.on("pot", (data) => {
      potStore.status = data;
    });

    wsService.on("vdf", (data) => {
      potStore.vdf = data;
    });

    wsService.on("system", (data) => {
      systemStore.overview = data;
    });
  } catch (err) {
    console.error("WebSocket connection failed, using polling only", err);
  }
});

onUnmounted(() => {
  // 清理定时器
  timers.forEach((timer) => clearInterval(timer));
  timers = [];

  // 断开 WebSocket
  wsService?.disconnect();
});
</script>

<style>
@import "./assets/main.css";

.app {
  font-family: "Inter", -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto,
    Oxygen, Ubuntu, Cantarell, sans-serif;
}
</style>
