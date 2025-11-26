<template>
  <div :id="chartId" class="circle-progress" :style="{ width: width, height: height }"></div>
</template>

<script setup lang="ts">
import { ref, onMounted, watch, onBeforeUnmount } from 'vue'
import * as echarts from 'echarts'

interface Props {
  value?: number
  title?: string
  color?: string
  backgroundColor?: string
  outerRingColor?: string
  textColor?: string
  width?: string
  height?: string
  innerRadius?: [number, number]
  showPercentSign?: boolean
  displayValue?: string
}

const props = withDefaults(defineProps<Props>(), {
  value: 0,
  title: '目前进度',
  color: 'rgba(0,153,255,0.8)',
  backgroundColor: 'rgba(0,153,255,0.1)',
  outerRingColor: 'rgba(0,153,255,0.3)',
  textColor: '#B7E1FF',
  width: '200px',
  height: '200px',
  innerRadius: () => [75, 90],
  showPercentSign: true,
  displayValue: ''
})

// 生成唯一ID
const chartId = ref(`circle-progress-${Math.random().toString(36).substr(2, 9)}`)
let chartInstance: echarts.ECharts | null = null

const initChart = () => {
  const chartDom = document.getElementById(chartId.value)
  if (!chartDom) return

  chartInstance = echarts.init(chartDom)
  updateChart()
}

const updateChart = () => {
  if (!chartInstance) return

  const option = {
    title: {
      text: props.title,
      subtext: props.displayValue || `${props.value}${props.showPercentSign ? '%' : ''}`,
      x: 'center',
      y: 'center',
      itemGap: 5,
      textStyle: {
        color: props.textColor,
        fontWeight: 'normal',
        fontFamily: '微软雅黑',
        fontSize: 14
      },
      subtextStyle: {
        color: props.textColor,
        fontWeight: 'bolder',
        fontSize: 16,
        fontFamily: '微软雅黑'
      }
    },
    series: [
      // 主进度环
      {
        type: 'pie',
        center: ['50%', '50%'],
        radius: props.innerRadius,
        x: '0%',
        tooltip: { show: false },
        hoverAnimation: false,
        data: [
          {
            name: '达成率',
            value: props.value,
            itemStyle: { color: props.color },
            label: { show: false },
            labelLine: { show: false }
          },
          {
            name: '未完成',
            value: 100 - props.value,
            itemStyle: { color: props.backgroundColor },
            label: { show: false },
            labelLine: { show: false }
          }
        ]
      },
      // 外环
      {
        type: 'pie',
        center: ['50%', '50%'],
        radius: [props.innerRadius[1] + 5, props.innerRadius[1] + 10],
        x: '0%',
        hoverAnimation: false,
        tooltip: { show: false },
        data: [
          {
            value: 100,
            itemStyle: { color: props.outerRingColor },
            label: { show: false },
            labelLine: { show: false }
          }
        ]
      },
      // 内环
      {
        type: 'pie',
        center: ['50%', '50%'],
        radius: [props.innerRadius[0] - 6, props.innerRadius[0] - 5],
        x: '0%',
        hoverAnimation: false,
        tooltip: { show: false },
        data: [
          {
            value: 100,
            itemStyle: { color: props.outerRingColor },
            label: { show: false },
            labelLine: { show: false }
          }
        ]
      }
    ]
  }

  chartInstance.setOption(option)
}

// 监听数据变化
watch(
  () => [props.value, props.title, props.color, props.textColor],
  () => {
    updateChart()
  },
  { deep: true }
)

onMounted(() => {
  initChart()
  // 监听窗口大小变化
  window.addEventListener('resize', () => {
    chartInstance?.resize()
  })
})

onBeforeUnmount(() => {
  chartInstance?.dispose()
  window.removeEventListener('resize', () => {
    chartInstance?.resize()
  })
})

// 暴露方法供父组件调用
defineExpose({
  resize: () => chartInstance?.resize(),
  getInstance: () => chartInstance
})
</script>

<style scoped>
.circle-progress {
  display: inline-block;
}
</style>
