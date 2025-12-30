<script setup>
import { onMounted, onBeforeUnmount, watch, ref, nextTick, computed } from 'vue'
import { Chart } from 'chart.js/auto'

Chart.defaults.maintainAspectRatio = false

const props = defineProps({
  labels: { type: Array, required: true }, // ISO strings
  series: { type: Array, default: null }, // numbers or nulls
  seriesList: { type: Array, default: () => [] }, // [{ label, data, color }]
  title: { type: String, default: '' },
  yLabel: { type: String, default: '' },
  segments: { type: Array, default: () => [] }, // [{ start, end, color, label }]
})

const canvas = ref(null)
let chart = null

const palette = ['#1976d2', '#c62828', '#2e7d32', '#ff9800', '#6a1b9a', '#00897b']

// Parse ISO -> elapsed seconds from first sample
const model = computed(() => {
  const labelCount = props.labels?.length ?? 0
  if (labelCount === 0) return { series: [], times: [], hasData: false }

  const inputs = []
  if (Array.isArray(props.series) && props.series.length) {
    inputs.push({ label: props.title || 'Series', data: props.series, color: palette[0] })
  }
  for (let i = 0; i < props.seriesList.length; i++) {
    const s = props.seriesList[i] || {}
    inputs.push({
      label: s.label || s.title || `Series ${i + 1}`,
      data: Array.isArray(s.data) ? s.data : [],
      color: s.color || palette[(i + 1) % palette.length],
    })
  }

  const times = new Array(labelCount)
  for (let i = 0; i < labelCount; i++) {
    const t = Date.parse(props.labels[i])
    times[i] = Number.isFinite(t) ? t / 1000 : NaN
  }
  const t0 = times.find((v) => Number.isFinite(v))

  const normalizeSeries = (values) => {
    const n = Math.min(labelCount, values.length)
    const points = []
    let hasData = false
    for (let i = 0; i < n; i++) {
      const y = values[i]
      const x = Number.isFinite(times[i]) && Number.isFinite(t0) ? times[i] - t0 : i
      const yy = y == null ? null : +y
      if (yy != null && Number.isFinite(yy)) hasData = true
      points.push({ x, y: yy })
    }
    return { points, hasData }
  }

  const normalized = inputs.map((s) => {
    const { points, hasData } = normalizeSeries(s.data || [])
    return { label: s.label, color: s.color, points, hasData }
  })
  const hasData = normalized.some((s) => s.hasData)
  return { series: normalized, times, hasData }
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

  const { series, hasData } = model.value
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
    const datasets = series.map((s, idx) => {
      const color = s.color || palette[idx % palette.length] || '#1976d2'
      return {
        label: s.label || undefined,
        data: s.points,
        borderWidth: 1,
        pointRadius: 0,
        spanGaps: true,
        borderColor: color,
        segment: {
          borderColor: (ctx) => {
            const mid = (ctx.p0?.parsed?.x + ctx.p1?.parsed?.x) / 2
            const match = segments.find((seg) => mid >= seg.start && mid <= seg.end)
            return match?.color || color
          },
        },
      }
    })

    chart = new Chart(ctx, {
      type: 'line',
      data: {
        datasets,
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
          legend: { display: datasets.length > 1 },
          decimation: { enabled: false }, // keep off until we prefilter
          tooltip: {
            mode: 'index',
            intersect: false,
            callbacks: {
              // Show human time on tooltip title
              title: (items) => fmtHMS(items?.[0]?.raw?.x ?? 0),
            },
          },
          segmentBands: { bands: props.segments },
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
  () => [props.labels, props.series, props.seriesList, props.segments],
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
