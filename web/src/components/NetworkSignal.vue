<template>
  <div class="network-signal flex items-end gap-1">
    <div 
      v-for="(bar, index) in bars" 
      :key="index"
      class="signal-bar"
      :class="{ active: index < activeLevel }"
      :style="{ height: bar.height }"
    ></div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

interface Props {
  level?: number // 0-4, 0=无信号, 4=满格
  status?: 'healthy' | 'warning' | 'error'
}

const props = withDefaults(defineProps<Props>(), {
  level: 4,
  status: 'healthy'
})

const bars = [
  { height: '6px' },
  { height: '10px' },
  { height: '14px' },
  { height: '18px' }
]

const activeLevel = computed(() => {
  if (props.status === 'error') return 1
  if (props.status === 'warning') return 2
  return props.level
})
</script>

<style scoped>
.signal-bar {
  width: 4px;
  background: rgba(255, 255, 255, 0.2);
  border-radius: 1px;
  transition: all 0.3s ease;
}

.signal-bar.active {
  background: linear-gradient(to top, #52c41a, #73d13d);
  box-shadow: 0 0 4px rgba(82, 196, 26, 0.5);
}

.network-signal:has(.signal-bar.active:nth-child(1):not(:nth-child(2).active)) .signal-bar.active {
  background: linear-gradient(to top, #ff4d4f, #ff7875);
  box-shadow: 0 0 4px rgba(255, 77, 79, 0.5);
}

.network-signal:has(.signal-bar.active:nth-child(2):not(:nth-child(3).active)) .signal-bar.active {
  background: linear-gradient(to top, #faad14, #ffc53d);
  box-shadow: 0 0 4px rgba(250, 173, 20, 0.5);
}
</style>
