<script setup>
import { ref, onMounted, computed } from 'vue'
import { useRoute } from 'vue-router'
import { useActivitiesStore } from '../stores/activities'
import TrackMap from '../components/TrackMap.vue'
import TimeSeriesChart from '../components/TimeSeriesChart.vue'
import { buildSegments, buildTrackPoints, toChartBands } from '../utils/segmentation'

const route = useRoute()
const id = route.params.id
const store = useActivitiesStore()

const detail = ref(null)
const loading = computed(() => store.loading)
const error = computed(() => store.error)
const coloring = ref('kilometers') // laps | kilometers | speed | hr

onMounted(async () => {
  try {
    detail.value = await store.fetchDetail(id)
  } catch {
    console.error(store.error)
  }
})

// conversions for speed series
function toKmh(arr) {
  return arr.map((v) => (v == null ? null : v * 3.6))
}
const paceSeries = computed(() => detail.value?.series?.pace_min_per_km || [])
const speedSeries = computed(() => {
  if (!detail.value) return []
  const src = detail.value.series.speed_mps
  const pace = paceSeries.value
  switch (store.unit) {
    case 'kmh':
      return toKmh(src)
    case 'pace':
      return pace
    default:
      return src
  }
})

const trackPoints = computed(() => buildTrackPoints(detail.value))

const coloredTrack = computed(() => buildSegments(trackPoints.value, coloring.value))

const chartSegments = computed(() => toChartBands(trackPoints.value, coloredTrack.value.segments))

function avgInRange(arr, start, end) {
  const vals = []
  for (let i = start; i <= end && i < arr.length; i++) {
    const v = arr[i]
    if (Number.isFinite(v)) vals.push(v)
  }
  if (!vals.length) return null
  return vals.reduce((a, b) => a + b, 0) / vals.length
}

function formatPace(minPerKm) {
  if (!Number.isFinite(minPerKm)) return null
  const totalSeconds = Math.round(minPerKm * 60)
  const minutes = Math.floor(totalSeconds / 60)
  const seconds = String(totalSeconds % 60).padStart(2, '0')
  return `${minutes}:${seconds}`
}

const segmentComparisons = computed(() => {
  if (!detail.value) return []
  const pace = paceSeries.value
  const hr = detail.value.series?.hr || []
  return (coloredTrack.value.segments || [])
    .map((seg, idx) => {
      const start = seg.startIdx ?? 0
      const end = seg.endIdx ?? start
      const avgPace = avgInRange(pace, start, end)
      const avgHr = avgInRange(hr, start, end)
      return {
        key: `${seg.label || 'Segment'}-${idx}`,
        label: seg.label || `Segment ${idx + 1}`,
        pace: formatPace(avgPace),
        hr: Number.isFinite(avgHr) ? Math.round(avgHr) : null,
      }
    })
    .filter((s) => s.pace != null || s.hr != null)
})
</script>

<template>
  <h2>Activity {{ id }}</h2>
  <p v-if="loading">Loading…</p>
  <p v-if="error" style="color: #b00">{{ error }}</p>

  <div v-if="detail">
    <p>
      Start: {{ new Date(detail.summary.started_at).toLocaleString() }} · Distance:
      {{ detail.summary.distance_m ?? 0 }} m · Duration: {{ detail.summary.duration_sec ?? 0 }} s
    </p>

    <!-- unit toggle -->
    <div style="margin: 8px 0; display: flex; gap: 8px; align-items: center">
      <span>Speed units:</span>
      <button :disabled="store.unit === 'mps'" @click="store.setUnit('mps')">m/s</button>
      <button :disabled="store.unit === 'kmh'" @click="store.setUnit('kmh')">km/h</button>
      <button :disabled="store.unit === 'pace'" @click="store.setUnit('pace')">min/km</button>
    </div>

    <div style="margin: 12px 0; display: flex; gap: 8px; align-items: center; flex-wrap: wrap">
      <span>Track coloring:</span>
      <button :disabled="coloring === 'kilometers'" @click="coloring = 'kilometers'">Kilometres</button>
      <button :disabled="coloring === 'laps'" @click="coloring = 'laps'">Laps</button>
      <button :disabled="coloring === 'speed'" @click="coloring = 'speed'">Speed</button>
      <button :disabled="coloring === 'hr'" @click="coloring = 'hr'">Heart rate</button>
    </div>

    <TrackMap
      :coords="detail.geojson.geometry.coordinates"
      :segments="coloredTrack.segments"
      :legend="coloredTrack.legend"
    />

    <section style="margin-top: 16px; display: grid; gap: 16px">
      <TimeSeriesChart
        :labels="detail.series.time_iso"
        :series="speedSeries"
        :segments="chartSegments"
        :title="
          store.unit === 'pace'
            ? 'Pace (min/km)'
            : store.unit === 'kmh'
              ? 'Speed (km/h)'
              : 'Speed (m/s)'
        "
        :y-label="store.unit === 'pace' ? 'min/km' : store.unit === 'kmh' ? 'km/h' : 'm/s'"
      />
      <TimeSeriesChart
        :labels="detail.series.time_iso"
        :series="detail.series.hr"
        :segments="chartSegments"
        title="Heart Rate"
        y-label="bpm"
      />
      <TimeSeriesChart
        :labels="detail.series.time_iso"
        :series="detail.series.elevation"
        :segments="chartSegments"
        title="Elevation"
        y-label="m"
      />
    </section>

    <section v-if="segmentComparisons.length" style="margin-top: 16px">
      <h3>Segment comparison</h3>
      <p style="color: #555">Averages are based on the current track coloring.</p>
      <table style="border-collapse: collapse; width: 100%; max-width: 640px">
        <thead>
          <tr>
            <th style="text-align: left; padding: 6px 8px; border-bottom: 1px solid #ccc">Segment</th>
            <th style="text-align: left; padding: 6px 8px; border-bottom: 1px solid #ccc">Pace (min/km)</th>
            <th style="text-align: left; padding: 6px 8px; border-bottom: 1px solid #ccc">Heart rate (bpm)</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="row in segmentComparisons" :key="row.key">
            <td style="padding: 6px 8px; border-bottom: 1px solid #eee">{{ row.label }}</td>
            <td style="padding: 6px 8px; border-bottom: 1px solid #eee">{{ row.pace ?? '—' }}</td>
            <td style="padding: 6px 8px; border-bottom: 1px solid #eee">{{ row.hr ?? '—' }}</td>
          </tr>
        </tbody>
      </table>
    </section>
  </div>
</template>
