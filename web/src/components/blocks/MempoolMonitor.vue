<template>
  <div class="mempool-monitor card h-full">
    <div class="box">
      <div class="tit">
        <span>交易池监控</span>
        <p></p>
      </div>
    </div>

    <a-spin :spinning="loading">
      <div class="space-y-4">
        <!-- 交易池总大小 -->
        <div>
          <div class="flex items-center justify-between mb-2">
            <div class="flex items-center gap-2">
              <span class="text-sm text-gray-400">交易池总大小</span>
              <span class="text-white font-semibold">
                {{ status?.totalSize || 0 }}
              </span>
            </div>
            <!-- 图例 -->
            <div class="flex items-center gap-3 text-xs">
              <div class="flex items-center gap-1">
                <div class="w-3 h-3 bg-green-500 rounded"></div>
                <span class="text-gray-400">已提议</span>
              </div>
              <div class="flex items-center gap-1">
                <div class="w-3 h-3 bg-yellow-500 rounded"></div>
                <span class="text-gray-400">待提议</span>
              </div>
            </div>
          </div>

          <!-- 进度条显示已提议/待提议 -->
          <div class="relative h-6 bg-gray-700 rounded-full overflow-hidden">
            <!-- 已提议部分 (绿色) -->
            <div
              class="absolute left-0 top-0 h-full bg-green-500 transition-all duration-300 flex items-center justify-center"
              :style="{ width: markedPercent + '%' }"
            >
              <span
                v-if="markedPercent > 10"
                class="text-xs text-white font-semibold"
              >
                {{ status?.markedTxs || 0 }}
              </span>
            </div>
            <!-- 待提议部分 (黄色) -->
            <div
              class="absolute h-full bg-yellow-500 transition-all duration-300 flex items-center justify-center"
              :style="{
                left: markedPercent + '%',
                width: unmarkedPercent + '%',
              }"
            >
              <span
                v-if="unmarkedPercent > 10"
                class="text-xs text-white font-semibold"
              >
                {{ status?.unmarkedTxs || 0 }}
              </span>
            </div>
          </div>
        </div>

        <!-- 交易类型分布 -->
        <div>
          <div class="text-sm text-gray-400 mb-1">交易类型分布</div>
          <div class="flex items-center gap-3">
            <!-- 3D饼图 -->
            <div class="flex-shrink-0" style="width: 140px">
              <TxTypePieChart
                :normal="status?.txTypes?.normal || 0"
                :bci="status?.txTypes?.bci || 0"
                :devastate="status?.txTypes?.devastate || 0"
              />
            </div>
            <!-- 图例 -->
            <div class="flex-1 flex flex-col justify-center space-y-1 text-sm">
              <div class="flex items-center gap-2">
                <div
                  class="w-2.5 h-2.5 rounded"
                  style="background-color: #4a9eff"
                ></div>
                <span>普通交易</span>
              </div>
              <div class="flex items-center gap-2">
                <div
                  class="w-2.5 h-2.5 rounded"
                  style="background-color: #ffd700"
                ></div>
                <span>BCI交易</span>
              </div>
              <div class="flex items-center gap-2">
                <div
                  class="w-2.5 h-2.5 rounded"
                  style="background-color: #ff4444"
                ></div>
                <span>Devastate</span>
              </div>
            </div>
          </div>
        </div>

        <!-- 平均确认时间 -->
        <div class="flex items-center justify-between">
          <span class="text-sm text-gray-400">平均确认时间</span>
          <span class="text-white"
            >{{ status?.avgConfirmTime?.toFixed(2) || "0.00" }}s</span
          >
        </div>

        <!-- 验证成功率 -->
        <!-- <div>
          <div class="flex items-center justify-between mb-1">
            <span class="text-sm text-gray-400">验证成功率</span>
            <span class="text-white">{{ status?.verifySuccessRate?.toFixed(1) || '0.0' }}%</span>
          </div>
          <a-progress
            :percent="status?.verifySuccessRate || 0"
            :stroke-color="{ '0%': '#ffa500', '100%': '#00ff88' }"
            :show-info="false"
          />
        </div> -->
      </div>
    </a-spin>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { storeToRefs } from "pinia";
import { DatabaseOutlined } from "@ant-design/icons-vue";
import { useMempoolStore } from "@/stores/mempool";
import TxTypePieChart from "./TxTypePieChart.vue";

const mempoolStore = useMempoolStore();
const { status, loading } = storeToRefs(mempoolStore);

// 计算已提议交易的百分比
const markedPercent = computed(() => {
  const total = status.value?.totalSize || 0;
  if (total === 0) return 0;
  const marked = status.value?.markedTxs || 0;
  return (marked / total) * 100;
});

// 计算待提议交易的百分比
const unmarkedPercent = computed(() => {
  const total = status.value?.totalSize || 0;
  if (total === 0) return 0;
  const unmarked = status.value?.unmarkedTxs || 0;
  return (unmarked / total) * 100;
});
</script>

<style scoped>
.space-y-4 > * + * {
  margin-top: 1rem;
}

.space-y-2 > * + * {
  margin-top: 0.5rem;
}
</style>
