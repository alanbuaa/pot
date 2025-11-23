<template>
  <div class="performance-metrics card h-full">
    <div class="box">
      <div class="tit">
        <span>性能指标</span>
        <p></p>
      </div>
    </div>

    <div class="space-y-4">
      <!-- TPS -->
      <div>
        <div class="text-sm text-gray-400 mb-1">实时 TPS</div>
        <div class="text-3xl font-bold text-pot-mining">
          {{ overview?.currentTPS?.toFixed(2) || "0.00" }}
        </div>
      </div>

      <!-- 平均出块时间 -->
      <div class="flex items-center justify-between">
        <span class="text-sm text-gray-400">平均出块时间</span>
        <span class="text-white font-semibold">
          {{ overview?.avgBlockTime?.toFixed(2) || "0.00" }}s
        </span>
      </div>

      <!-- 交易池大小 -->
      <div class="flex items-center justify-between">
        <span class="text-sm text-gray-400">交易池大小</span>
        <a-tag color="blue">{{ overview?.mempoolSize || 0 }}</a-tag>
      </div>

      <!-- 网络利用率 -->
      <div>
        <div class="flex items-center justify-between mb-1">
          <span class="text-sm text-gray-400">网络利用率</span>
          <span class="text-white"
            >{{ overview?.networkUtilization || 0 }}%</span
          >
        </div>
        <a-progress
          :percent="overview?.networkUtilization || 0"
          :stroke-color="getUtilizationColor(overview?.networkUtilization || 0)"
          :show-info="false"
        />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { storeToRefs } from "pinia";
import { DashboardOutlined } from "@ant-design/icons-vue";
import { useSystemStore } from "@/stores/system";

const systemStore = useSystemStore();
const { overview } = storeToRefs(systemStore);

function getUtilizationColor(utilization: number) {
  if (utilization >= 80) return "#ff4444";
  if (utilization >= 60) return "#ffaa00";
  return "#00ff88";
}
</script>

<style scoped>
.space-y-4 > * + * {
  margin-top: 1rem;
}
</style>
