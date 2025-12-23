<script setup>
import { ref, onMounted, computed } from 'vue'
import { useRoute } from 'vue-router'
import { useActivitiesStore } from '../stores/activities'
import TrackMap from '../components/TrackMap.vue'
import TimeSeriesChart from '../components/TimeSeriesChart.vue'

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
function toPaceMinPerKm(arr) {
  // m/s -> s/m -> s/km -> min/km
  return arr.map((v) => {
    if (v == null || v <= 0) return null
    const secPerKm = 1000 / v
    const m = Math.floor(secPerKm / 60)
    const s = Math.round(secPerKm % 60)
    return Number(`${m}.${String(s).padStart(2, '0')}`) // e.g., 5.12 meaning 5'12"
  })
}
const speedSeries = computed(() => {
  if (!detail.value) return []
  const src = detail.value.series.speed_mps
  switch (store.unit) {
    case 'kmh':
      return toKmh(src)
    case 'pace':
      return toPaceMinPerKm(src) // “min.km” formatting explained below
    default:
      return src
  }
})

// Helpers for map coloring
const palette = ['#2c7bb6', '#00a6ca', '#00ccbc', '#90eb9d', '#ffff8c', '#f9d057', '#f29e2e', '#e76818', '#d7191c', '#9e0142']

function haversineMeters(lat1, lon1, lat2, lon2) {
  const toRad = (d) => (d * Math.PI) / 180
  const R = 6371000
  const dLat = toRad(lat2 - lat1)
  const dLon = toRad(lon2 - lon1)
  const a =
    Math.sin(dLat / 2) * Math.sin(dLat / 2) +
    Math.cos(toRad(lat1)) * Math.cos(toRad(lat2)) * Math.sin(dLon / 2) * Math.sin(dLon / 2)
  const c = 2 * Math.atan2(Math.sqrt(a), Math.sqrt(1 - a))
  return R * c
}

const trackPoints = computed(() => {
  if (!detail.value) return []
  const coords = detail.value.geojson?.geometry?.coordinates || []
  const times = detail.value.series?.time_iso || []
  const hr = detail.value.series?.hr || []
  const speed = detail.value.series?.speed_mps || []

  let total = 0
  const pts = []
  for (let i = 0; i < coords.length; i++) {
    const [lon, lat] = coords[i]
    if (i > 0) {
      const [prevLon, prevLat] = coords[i - 1]
      total += haversineMeters(prevLat, prevLon, lat, lon)
    }
    pts.push({ lon, lat, t: times?.[i], hr: hr?.[i], speed: speed?.[i], dist: total })
  }
  return pts
})

function quantile(sortedVals, q) {
  if (!sortedVals.length) return 0
  const idx = Math.max(0, Math.min(sortedVals.length - 1, Math.floor((sortedVals.length - 1) * q)))
  return sortedVals[idx]
}

function buildBucketSegments(points, valueFn, labels) {
  const values = points.map(valueFn).filter((v) => Number.isFinite(v))
  if (!values.length) return { segments: [], legend: [] }

  values.sort((a, b) => a - b)
  const thresholds = [quantile(values, 0.25), quantile(values, 0.5), quantile(values, 0.75)]

  const legend = []
  const segments = []

  let currentBucket = null
  let currentCoords = []
  let currentStartIdx = null
  for (let i = 0; i < points.length; i++) {
    const v = valueFn(points[i])
    let bucket = -1
    if (Number.isFinite(v)) {
      if (v <= thresholds[0]) bucket = 0
      else if (v <= thresholds[1]) bucket = 1
      else if (v <= thresholds[2]) bucket = 2
      else bucket = 3
    }

    const coord = [points[i].lon, points[i].lat]
    if (bucket !== currentBucket) {
      if (currentCoords.length > 1 && currentBucket != null && currentBucket >= 0) {
        const color = palette[currentBucket]
        segments.push({ coords: currentCoords, color, label: labels[currentBucket], startIdx: currentStartIdx, endIdx: i - 1 })
        if (!legend.some((l) => l.label === labels[currentBucket])) legend.push({ color, label: labels[currentBucket] })
      }
      currentCoords = bucket >= 0 ? [coord] : []
      currentStartIdx = bucket >= 0 ? i : null
      currentBucket = bucket
    } else if (bucket >= 0) {
      currentCoords.push(coord)
    }
  }
  if (currentCoords.length > 1 && currentBucket != null && currentBucket >= 0) {
    const color = palette[currentBucket]
    segments.push({ coords: currentCoords, color, label: labels[currentBucket], startIdx: currentStartIdx, endIdx: points.length - 1 })
    if (!legend.some((l) => l.label === labels[currentBucket])) legend.push({ color, label: labels[currentBucket] })
  }
  return { segments, legend }
}

function buildBoundarySegments(points, boundaries, labelForIdx) {
  const segments = []
  const legend = []
  for (let i = 0; i < boundaries.length - 1; i++) {
    const start = boundaries[i]
    const end = boundaries[i + 1]
    if (end <= start) continue
    const coords = points.slice(start, end + 1).map((p) => [p.lon, p.lat])
    if (coords.length < 2) continue
    const color = palette[i % palette.length]
    const label = labelForIdx(i)
    segments.push({ coords, color, label, startIdx: start, endIdx: end })
    legend.push({ color, label })
  }
  return { segments, legend }
}

function chartBands(points, segments) {
  if (!points.length || !segments.length) return []
  const times = points.map((p) => Date.parse(p.t)).map((t) => (Number.isFinite(t) ? t / 1000 : NaN))
  const t0 = times.find((v) => Number.isFinite(v))
  const fallbackIdx = (idx) => idx
  const baseline = Number.isFinite(t0) ? t0 : null

  const xValue = (idx) => {
    if (!Number.isInteger(idx) || idx < 0 || idx >= times.length) return NaN
    if (baseline != null && Number.isFinite(times[idx])) return times[idx] - baseline
    return fallbackIdx(idx)
  }

  return segments
    .map((seg) => {
      const start = xValue(seg.startIdx)
      const end = xValue(seg.endIdx)
      if (!Number.isFinite(start) || !Number.isFinite(end) || end <= start) return null
      return { start, end, color: seg.color, label: seg.label }
    })
    .filter(Boolean)
}

const coloredTrack = computed(() => {
  const pts = trackPoints.value
  if (!pts.length) return { segments: [], legend: [] }

  if (coloring.value === 'kilometers') {
    const bounds = [0]
    let next = 1000
    for (let i = 1; i < pts.length; i++) {
      if (pts[i].dist >= next) {
        bounds.push(i)
        next += 1000
      }
    }
    bounds.push(pts.length - 1)
    return buildBoundarySegments(pts, bounds, (i) => `Km ${i + 1}`)
  }

  if (coloring.value === 'laps') {
    const bounds = [0]
    const lapMinDist = 150 // meters to avoid noise
    const startLat = pts[0].lat
    const startLon = pts[0].lon
    for (let i = 1; i < pts.length - 1; i++) {
      const sinceLap = pts[i].dist - pts[bounds[bounds.length - 1]].dist
      const distFromStart = haversineMeters(startLat, startLon, pts[i].lat, pts[i].lon)
      if (sinceLap >= lapMinDist && distFromStart < 25) {
        bounds.push(i)
      }
    }
    if (bounds[bounds.length - 1] !== pts.length - 1) bounds.push(pts.length - 1)
    return buildBoundarySegments(pts, bounds, (i) => `Lap ${i + 1}`)
  }

  if (coloring.value === 'speed') {
    const labels = ['Very slow', 'Easy', 'Moderate', 'Fastest']
    return buildBucketSegments(pts, (p) => (p.speed == null ? NaN : p.speed * 3.6), labels)
  }

  if (coloring.value === 'hr') {
    const labels = ['Low HR', 'Aerobic', 'Tempo', 'Max effort']
    return buildBucketSegments(pts, (p) => (p.hr == null ? NaN : p.hr), labels)
  }

  return { segments: [], legend: [] }
})

const chartSegments = computed(() => chartBands(trackPoints.value, coloredTrack.value.segments))
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
  </div>
</template>
