<template>
  <div class="top-bar px-6 relative">
    <!-- 指标区域 - 放在上半部分 -->
    <div class="metrics-container">
      <div class="flex items-center gap-8 justify-around">
        <div class="stat-item-large"></div>

        <!-- 平均出块时间 -->
        <div class="stat-item-large">
          <div class="stat-label">平均出块时间</div>
          <div class="stat-value">
            {{ overview?.avgBlockTime?.toFixed(2) || '0.00' }}s
          </div>
        </div>

        <!-- 区块高度 -->
        <div class="stat-item-large">
          <div class="stat-label">区块高度</div>
          <div class="stat-value text-committee-leader">
            {{ formatNumber(overview?.currentHeight || 0) }}
          </div>
        </div>

        <!-- 当前TPS -->
        <div class="stat-item-large">
          <!-- <div class="stat-label">实时TPS</div>
          <div class="stat-value text-pot-mining">
            {{ overview?.currentTPS?.toFixed(2) || '0.00' }}
          </div> -->
        </div>

        <div class="stat-item-large"></div>
        <div class="stat-item-large"></div>
        <div class="stat-item-large"></div>
        <div class="stat-item-large"></div>
        <div class="stat-item-large"></div>
        <div class="stat-item-large"></div>
  <div class="stat-item-large"></div>
        <div class="stat-item-large"></div>

        <!-- 节点状态 -->
        <div class="stat-item-large">
          <div class="stat-label">节点状态</div>
          <div class="stat-value flex items-center gap-2">
            <span :class="['status-dot', getNetworkStatusClass()]"></span>
            {{ overview?.onlineNodes || 0 }}/{{ overview?.totalNodes || 0 }}
          </div>
        </div>

        <!-- 网络健康度 - 信号格 -->
        <div class="stat-item-large">
          <div class="stat-label">网络健康</div>
          <div class="stat-value flex items-center justify-center">
            <NetworkSignal
              :level="getSignalLevel()"
              :status="getNetworkStatus()"
            />
          </div>
        </div>
        <div class="stat-item-large"></div>

      </div>
    </div>

    <!-- 标题 - 居中显示 -->
    <div class="title-area">
      <h1 class="main-title">POT共识状态可视化</h1>
    </div>
  </div>
</template>

<script setup lang="ts">
import { storeToRefs } from "pinia";
import { useSystemStore } from "@/stores/system";
import { formatNumber, formatDuration } from "@/utils/format";
import NetworkSignal from "@/components/NetworkSignal.vue";

const systemStore = useSystemStore();
const { overview } = storeToRefs(systemStore);

function getNetworkStatusClass() {
  const status = overview.value?.networkStatus;
  if (status === "healthy") return "online";
  if (status === "warning") return "warning";
  return "error";
}

function getSignalLevel(): number {
  const utilization = overview.value?.networkUtilization || 0;
  if (utilization >= 90) return 4;
  if (utilization >= 70) return 3;
  if (utilization >= 50) return 2;
  if (utilization >= 30) return 1;
  return 0;
}

function getNetworkStatus(): "healthy" | "warning" | "error" {
  const status = overview.value?.networkStatus;
  if (status === "healthy") return "healthy";
  if (status === "warning") return "warning";
  return "error";
}
</script>

<style scoped>
.top-bar {
  background-image: url("@/assets/images/topbg.png");
  background-size: 100% 100%;
  background-position: center top;
  background-repeat: no-repeat;
  height: 80px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.metrics-container {
  position: absolute;
  top: 8px;
  left: 0;
  right: 0;
  padding: 0 24px;
}

.title-area {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
}

.main-title {
  font-size: 28px;
  font-weight: 900;
  color: #ffffff;
  text-shadow: 0 2px 8px rgba(0, 0, 0, 0.5);
  letter-spacing: 2px;
  white-space: nowrap;
}

.stat-item-large {
  display: flex;
  flex-direction: row;
  align-items: center;
  gap: 8px;
}

.stat-label {
  font-size: 12px;
  color: rgba(255, 255, 255, 0.7);
  white-space: nowrap;
}

.stat-value {
  font-size: 14px;
  font-weight: 600;
  color: #ffffff;
  display: flex;
  align-items: center;
  gap: 4px;
}

@media (max-width: 1280px) {
  .stat-item-compact {
    min-width: 70px;
  }

  .stat-label-compact {
    font-size: 10px;
  }

  .stat-value-compact {
    font-size: 13px;
  }
}
</style>
