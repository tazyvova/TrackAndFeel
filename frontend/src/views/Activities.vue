<script setup>
import { ref, onMounted } from 'vue'

const items = ref([])
const loading = ref(false)
const error = ref('')

async function load() {
  loading.value = true
  error.value = ''
  try {
    const res = await fetch('/api/activities?limit=20')
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    const data = await res.json()
    items.value = data.items || []
  } catch (e) {
    error.value = String(e)
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>

<template>
  <h2>Activities</h2>
  <p v-if="loading">Loading…</p>
  <p v-if="error" style="color: #b00">{{ error }}</p>
  <ul v-if="items.length">
    <li v-for="a in items" :key="a.id" style="margin: 6px 0">
      <router-link :to="`/activities/${a.id}`">
        {{ new Date(a.started_at).toLocaleString() }}
      </router-link>
      — {{ a.distance_m ?? 0 }} m
      <span v-if="a.avg_hr"> — avg HR {{ a.avg_hr }}</span>
    </li>
  </ul>
  <p v-else-if="!loading">No activities yet.</p>
</template>
