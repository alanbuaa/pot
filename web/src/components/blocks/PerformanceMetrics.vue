<template>
  <div class="performance-metrics card h-full">
    <div class="box">
      <div class="tit">
        <span>性能指标</span>
        <p></p>
      </div>
    </div>

    <div class="space-y-2">
      <!-- 交易池大小 -->
      <!-- <div class="flex items-center justify-between">
        <span class="text-sm text-gray-400">交易池大小</span>
        <a-tag color="blue">{{ overview?.mempoolSize || 0 }}</a-tag>
      </div> -->

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
.space-y-2 > * + * {
  margin-top: 0.5rem;
}
</style>
