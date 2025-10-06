<script setup>
import { onMounted, onBeforeUnmount, watch, ref } from 'vue'
import L from 'leaflet'
import 'leaflet/dist/leaflet.css'

const props = defineProps({
  coords: { type: Array, required: true }, // [[lon,lat], ...] from /track geojson
})

let map, layer
const mapEl = ref(null)

onMounted(() => {
  map = L.map(mapEl.value, { zoomControl: true })
  L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
    maxZoom: 19,
    attribution: '&copy; OpenStreetMap',
  }).addTo(map)

  draw()
})

watch(() => props.coords, draw, { deep: true })

function draw() {
  if (!map || !props.coords?.length) return
  if (layer) layer.remove()

  // convert [lon,lat] -> [lat,lon] for Leaflet
  const latlngs = props.coords.map(([lon, lat]) => [lat, lon])
  layer = L.polyline(latlngs, { weight: 4 }).addTo(map)
  map.fitBounds(layer.getBounds(), { padding: [20, 20] })
}

onBeforeUnmount(() => map && map.remove())
</script>

<template>
  <div
    ref="mapEl"
    style="height: 360px; width: 100%; border: 1px solid #eee; border-radius: 8px"
  ></div>
</template>
