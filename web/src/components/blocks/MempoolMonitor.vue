<template>
  <div class="mempool-monitor card h-full">
    <div class="box">
      <div class="tit">
        <span>交易池监控</span>
        <p></p>
      </div>
    </div>

    <a-spin :spinning="loading">
      <div class="space-y-2">
        <!-- 总大小 -->
        <div>
          <div class="flex items-center justify-between mb-2">
            <div class="flex items-center gap-2">
              <span class="text-sm text-gray-400">总大小</span>
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

        <!-- 类型分布 -->
        <div>
          <div class="text-sm text-gray-400 mb-2 mt-3">确认时间与类型分布</div>
          <div class="grid grid-cols-2 gap-3">
            <!-- 左侧：平均确认时间 -->
            <div class="flex items-center justify-center">
              <CircleProgress
                :value="normalizeConfirmTime(status?.avgConfirmTime || 0)"
                :displayValue="(status?.avgConfirmTime || 0).toFixed(2) + 's'"
                title="平均确认时间"
                :showPercentSign="false"
                color="rgba(139, 92, 246, 0.8)"
                backgroundColor="rgba(139, 92, 246, 0.1)"
                outerRingColor="rgba(139, 92, 246, 0.3)"
                width="120px"
                height="120px"
                :innerRadius="[40, 48]"
              />
            </div>
            
            <!-- 右侧：类型分布 -->
            <div class="flex items-center gap-2">
              <!-- 3D饼图 -->
              <div class="flex-shrink-0" style="width: 100px">
                <TxTypePieChart
                  :normal="status?.txTypes?.normal || 0"
                  :bci="status?.txTypes?.bci || 0"
                  :devastate="status?.txTypes?.devastate || 0"
                />
              </div>
              <!-- 图例 -->
              <div class="flex-1 flex flex-col justify-center space-y-1 text-xs">
                <div class="flex items-center gap-1.5">
                  <div
                    class="w-2 h-2 rounded"
                    style="background-color: #4a9eff"
                  ></div>
                  <span>普通交易</span>
                </div>
                <div class="flex items-center gap-1.5">
                  <div
                    class="w-2 h-2 rounded"
                    style="background-color: #ffd700"
                  ></div>
                  <span>BCI交易</span>
                </div>
                <div class="flex items-center gap-1.5">
                  <div
                    class="w-2 h-2 rounded"
                    style="background-color: #ff4444"
                  ></div>
                  <span>Devastate</span>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- 实时 TPS -->
        <div class="flex items-center justify-between">
          <span class="text-sm text-gray-400">实时 TPS</span>
          <span class="text-2xl font-bold text-pot-mining">
            {{ overview?.currentTPS?.toFixed(2) || "0.00" }}
          </span>
        </div>
      </div>
    </a-spin>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { storeToRefs } from "pinia";
import { DatabaseOutlined } from "@ant-design/icons-vue";
import { useMempoolStore } from "@/stores/mempool";
import { useSystemStore } from "@/stores/system";
import TxTypePieChart from "./TxTypePieChart.vue";
import CircleProgress from "@/components/CircleProgress.vue";

const mempoolStore = useMempoolStore();
const { status, loading } = storeToRefs(mempoolStore);
const systemStore = useSystemStore();
const { overview } = storeToRefs(systemStore);

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

// 将确认时间标准化到0-100范围 (假设最大10秒)
function normalizeConfirmTime(time: number): number {
  const maxTime = 10;
  return Math.min(100, Math.max(0, (time / maxTime) * 100));
}
</script>

<style scoped>
.space-y-2 > * + * {
  margin-top: 0.5rem;
}
</style>
