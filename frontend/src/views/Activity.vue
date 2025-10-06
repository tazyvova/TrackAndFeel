<script setup>
import { ref, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import TrackMap from '../components/TrackMap.vue'
import TimeSeriesChart from '../components/TimeSeriesChart.vue'

const route = useRoute()
const id = route.params.id
const detail = ref(null)
const loading = ref(false)
const error = ref('')

async function load() {
  loading.value = true
  error.value = ''
  try {
    const res = await fetch(`/api/activities/${id}/track`)
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    detail.value = await res.json()
  } catch (e) {
    error.value = String(e)
  } finally {
    loading.value = false
  }
}
onMounted(load)
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

    <TrackMap :coords="detail.geojson.geometry.coordinates" />
    <section v-if="detail" style="margin-top: 16px; display: grid; gap: 16px">
      <TimeSeriesChart
        :labels="detail.series.time_iso"
        :series="detail.series.speed_mps"
        title="Speed (m/s)"
        yLabel="m/s"
      />
      <TimeSeriesChart
        :labels="detail.series.time_iso"
        :series="detail.series.hr"
        title="Heart Rate"
        yLabel="bpm"
      />
      <TimeSeriesChart
        :labels="detail.series.time_iso"
        :series="detail.series.elevation"
        title="Elevation"
        yLabel="m"
      />
    </section>
  </div>
</template>
