<script setup>
import { onMounted } from 'vue'
import { useActivitiesStore } from '../stores/activities'

const store = useActivitiesStore()

onMounted(() => {
  if (!store.items.length) store.fetchList(20, 0)
})
</script>

<template>
  <h2>Activities</h2>
  <p v-if="store.loading">Loading…</p>
  <p v-if="store.error" style="color: #b00">{{ store.error }}</p>

  <ul v-if="store.items.length">
    <li v-for="a in store.items" :key="a.id" style="margin: 6px 0">
      <router-link :to="`/activities/${a.id}`">
        {{ new Date(a.started_at).toLocaleString() }}
      </router-link>
      — {{ a.distance_m ?? 0 }} m
      <span v-if="a.avg_hr"> — avg HR {{ a.avg_hr }}</span>
    </li>
  </ul>
  <p v-else-if="!store.loading">No activities yet.</p>
</template>
