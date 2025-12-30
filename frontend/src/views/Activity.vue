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
const speedSeries = computed(() => {
  if (!detail.value) return []
  const src = detail.value.series.speed_mps
  const pace = detail.value.series.pace_min_per_km
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
