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

    <TrackMap :coords="detail.geojson.geometry.coordinates" />

    <section style="margin-top: 16px; display: grid; gap: 16px">
      <TimeSeriesChart
        :labels="detail.series.time_iso"
        :series="speedSeries"
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
        title="Heart Rate"
        y-label="bpm"
      />
      <TimeSeriesChart
        :labels="detail.series.time_iso"
        :series="detail.series.elevation"
        title="Elevation"
        y-label="m"
      />
    </section>
  </div>
</template>
