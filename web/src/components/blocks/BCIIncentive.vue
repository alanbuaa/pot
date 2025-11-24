<template>
  <div class="bci-incentive card h-full">
    <div class="box">
      <div class="tit">
        <span>BCI 激励</span>
        <p></p>
      </div>
    </div>

    <a-spin :spinning="loading">
      <div class="space-y-2">
        <!-- 总奖励 -->
        <div class="flex items-center justify-between">
          <span class="text-sm text-gray-400">总奖励金额</span>
          <span class="text-xl font-bold text-incentive">
            {{ status?.totalReward?.toLocaleString() || 0 }}
            <span class="text-gray-400 mx-1">/</span>
            <a-tooltip title="累计利息">
              <span class="text-base text-white cursor-help">
                {{ status?.totalInterest?.toLocaleString() || 0 }}
              </span>
            </a-tooltip>
          </span>
        </div>

        <!-- 锁定激励和待分配奖励 -->
        <div class="grid grid-cols-2 gap-3">
          <!-- 锁定激励 -->
          <div class="flex items-center justify-center">
            <CircleProgress
              :value="
                normalizeValue(
                  status?.lockedReward || 0,
                  0,
                  status?.totalReward || 100000
                )
              "
              :displayValue="(status?.lockedReward || 0).toLocaleString()"
              title="锁定激励"
              :showPercentSign="false"
              color="rgba(34, 197, 94, 0.8)"
              backgroundColor="rgba(34, 197, 94, 0.1)"
              outerRingColor="rgba(34, 197, 94, 0.3)"
              width="120px"
              height="120px"
              :innerRadius="[40, 48]"
            />
          </div>

          <!-- 待分配奖励 -->
          <div class="flex items-center justify-center">
            <CircleProgress
              :value="normalizeValue(status?.pendingRewards || 0, 0, 1000)"
              :displayValue="(status?.pendingRewards || 0).toString()"
              title="待分配奖励"
              :showPercentSign="false"
              color="rgba(234, 179, 8, 0.8)"
              backgroundColor="rgba(234, 179, 8, 0.1)"
              outerRingColor="rgba(234, 179, 8, 0.3)"
              width="120px"
              height="120px"
              :innerRadius="[40, 48]"
            />
          </div>
        </div>

        <!-- 奖励分配比例 -->
        <div>
          <!-- <div class="text-sm text-gray-400 mb-1">奖励分配</div> -->
          <div class="flex items-center gap-3">
            <!-- 3D饼图 -->
            <div class="flex-shrink-0" style="width: 140px">
              <RewardRatioPieChart
                :miner="status?.rewardRatio?.miner || 0"
                :exchequer="status?.rewardRatio?.exchequer || 0"
                :committeeLeader="status?.rewardRatio?.committeeLeader || 0"
                :committeeMember="status?.rewardRatio?.committeeMember || 0"
              />
            </div>
            <!-- 图例 -->
            <div class="flex-1 flex flex-col justify-center space-y-1 text-xs">
              <div class="flex items-center justify-between">
                <div class="flex items-center gap-2">
                  <div
                    class="w-2.5 h-2.5 rounded"
                    style="background-color: #4a9eff"
                  ></div>
                  <span>矿工</span>
                </div>
                <span>{{ status?.rewardRatio?.miner || 0 }}%</span>
              </div>
              <div class="flex items-center justify-between">
                <div class="flex items-center gap-2">
                  <div
                    class="w-2.5 h-2.5 rounded"
                    style="background-color: #ff6b6b"
                  ></div>
                  <span>国库</span>
                </div>
                <span>{{ status?.rewardRatio?.exchequer || 0 }}%</span>
              </div>
              <div class="flex items-center justify-between">
                <div class="flex items-center gap-2">
                  <div
                    class="w-2.5 h-2.5 rounded"
                    style="background-color: #ffd700"
                  ></div>
                  <span>委员会Leader</span>
                </div>
                <span>{{ status?.rewardRatio?.committeeLeader || 0 }}%</span>
              </div>
              <div class="flex items-center justify-between">
                <div class="flex items-center gap-2">
                  <div
                    class="w-2.5 h-2.5 rounded"
                    style="background-color: #51cf66"
                  ></div>
                  <span>委员会成员</span>
                </div>
                <span>{{ status?.rewardRatio?.committeeMember || 0 }}%</span>
              </div>
            </div>
          </div>
          <!-- 累计利息 -->
          <!-- <div class="flex items-center justify-between">
            <span class="text-sm text-gray-400">累计利息</span>
            <span class="text-white">{{
              status?.totalInterest?.toLocaleString() || 0
            }}</span>
          </div> -->
        </div>
      </div>
    </a-spin>
  </div>
</template>

<script setup lang="ts">
import { storeToRefs } from "pinia";
import { DollarOutlined } from "@ant-design/icons-vue";
import { useBCIStore } from "@/stores/bci";
import RewardRatioPieChart from "./RewardRatioPieChart.vue";
import CircleProgress from "@/components/CircleProgress.vue";

const bciStore = useBCIStore();
const { status, loading } = storeToRefs(bciStore);

// 将值标准化到0-100范围
function normalizeValue(value: number, min: number, max: number): number {
  if (max === 0) return 0;
  return Math.min(100, Math.max(0, ((value - min) / (max - min)) * 100));
}
</script>

<style scoped>
.space-y-2 > * + * {
  margin-top: 0.5rem;
}

.space-y-1 > * + * {
  margin-top: 0.25rem;
}
</style>
