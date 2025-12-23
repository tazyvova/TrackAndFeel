<script setup>
import { onMounted, onBeforeUnmount, watch, ref, nextTick, computed } from 'vue'
import { Chart } from 'chart.js/auto'

Chart.defaults.maintainAspectRatio = false

const props = defineProps({
  labels: { type: Array, required: true }, // ISO strings
  series: { type: Array, required: true }, // numbers or nulls
  title: { type: String, default: '' },
  yLabel: { type: String, default: '' },
  segments: { type: Array, default: () => [] }, // [{ start, end, color, label }]
})

const canvas = ref(null)
let chart = null

// Parse ISO -> elapsed seconds from first sample
const model = computed(() => {
  const n = Math.min(props.labels.length, props.series.length)
  if (n === 0) return { points: [], times: [], hasData: false }

  const times = new Array(n)
  for (let i = 0; i < n; i++) {
    const t = Date.parse(props.labels[i])
    times[i] = Number.isFinite(t) ? t / 1000 : NaN
  }
  // baseline
  const t0 = times.find((v) => Number.isFinite(v))
  const points = []
  let hasData = false
  for (let i = 0; i < n; i++) {
    const y = props.series[i]
    const x = Number.isFinite(times[i]) && Number.isFinite(t0) ? times[i] - t0 : i
    const yy = y == null ? null : +y
    if (yy != null && Number.isFinite(yy)) hasData = true
    points.push({ x, y: yy })
  }
  return { points, times, hasData }
})

function fmtHMS(sec) {
  if (!Number.isFinite(sec)) return ''
  sec = Math.max(0, Math.round(sec))
  const h = Math.floor(sec / 3600)
  const m = Math.floor((sec % 3600) / 60)
  const s = sec % 60
  return h > 0
    ? `${h}:${String(m).padStart(2, '0')}:${String(s).padStart(2, '0')}`
    : `${m}:${String(s).padStart(2, '0')}`
}

const bandsPlugin = {
  id: 'segmentBands',
  beforeDraw(chart, _, opts) {
    const bands = opts?.bands
    if (!bands?.length) return
    const {
      ctx,
      chartArea: { top, bottom },
      scales: { x },
    } = chart
    ctx.save()
    for (const band of bands) {
      const start = Number(band.start)
      const end = Number(band.end)
      if (!Number.isFinite(start) || !Number.isFinite(end) || end <= start) continue
      const x1 = x.getPixelForValue(start)
      const x2 = x.getPixelForValue(end)
      ctx.fillStyle = band.color || '#999'
      ctx.globalAlpha = 0.08
      ctx.fillRect(x1, top, x2 - x1, bottom - top)
    }
    ctx.restore()
  },
}

function build() {
  if (chart) {
    chart.destroy()
    chart = null
  }

  const el = canvas.value
  if (!el) return
  const ctx = el.getContext('2d')
  if (!ctx) return

  const { points, hasData } = model.value
  const segments = (props.segments || [])
    .map((s) => ({ start: Number(s.start), end: Number(s.end), color: s.color || '#999', label: s.label }))
    .filter((s) => Number.isFinite(s.start) && Number.isFinite(s.end) && s.end > s.start)

  // If no finite Y values, show “No data” overlay and don’t instantiate Chart.js
  if (!hasData) {
    ctx.clearRect(0, 0, el.width, el.height)
    ctx.font = '14px system-ui, sans-serif'
    ctx.fillStyle = '#888'
    ctx.textAlign = 'center'
    ctx.fillText('No data', el.width / 2, el.height / 2)
    return
  }

  try {
    chart = new Chart(ctx, {
      type: 'line',
      data: {
        datasets: [
          {
            label: props.title || undefined,
            data: points, // [{x, y}]
            borderWidth: 1,
            pointRadius: 0,
            spanGaps: true,
            borderColor: '#1976d2',
            segment: {
              borderColor: (ctx) => {
                const mid = (ctx.p0?.parsed?.x + ctx.p1?.parsed?.x) / 2
                const match = segments.find((s) => mid >= s.start && mid <= s.end)
                return match?.color || '#1976d2'
              },
            },
          },
        ],
      },
      options: {
        responsive: true,
        animation: false,
        parsing: true,
        normalized: true,
        scales: {
          x: {
            type: 'linear',
            ticks: {
              maxTicksLimit: 8,
              callback: (v) => fmtHMS(v),
            },
            title: { display: true, text: 'time' },
          },
          y: {
            title: { display: !!props.yLabel, text: props.yLabel },
          },
        },
        plugins: {
          legend: { display: false },
          decimation: { enabled: false }, // keep off until we prefilter
          tooltip: {
            mode: 'index',
            intersect: false,
            callbacks: {
              // Show human time on tooltip title
              title: (items) => fmtHMS(items?.[0]?.raw?.x ?? 0),
            },
          },
          segmentBands: { bands: segments },
        },
        elements: { line: { tension: 0 } },
      },
      plugins: [bandsPlugin],
    })
  } catch (e) {
    console.error('TimeSeriesChart: failed to create chart', e)
  }
}

onMounted(async () => {
  await nextTick()
  build()
})
watch(
  () => [props.labels, props.series, props.segments],
  () => {
    nextTick().then(build)
  },
  { deep: true }
)
onBeforeUnmount(() => {
  if (chart) chart.destroy()
})
</script>

<template>
  <div style="height: 180px; width: 100%">
    <canvas ref="canvas" style="height: 100%; width: 100%"></canvas>
  </div>
</template>
