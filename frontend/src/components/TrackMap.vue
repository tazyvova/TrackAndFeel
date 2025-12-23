<script setup>
import { onMounted, onBeforeUnmount, watch, ref } from 'vue'
import L from 'leaflet'
import 'leaflet/dist/leaflet.css'

const props = defineProps({
  coords: { type: Array, required: true }, // [[lon,lat], ...] from /track geojson
  segments: { type: Array, default: () => [] }, // [{ coords, color?, label? }]
  legend: { type: Array, default: () => [] }, // [{ color, label }]
})

let map, baseLayer
let segmentLayers = []
const mapEl = ref(null)

onMounted(() => {
  map = L.map(mapEl.value, { zoomControl: true })
  L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
    maxZoom: 19,
    attribution: '&copy; OpenStreetMap',
  }).addTo(map)

  draw()
})

watch(
  () => [props.coords, props.segments],
  () => draw(),
  { deep: true }
)

function draw() {
  if (!map || !props.coords?.length) return
  if (baseLayer) {
    baseLayer.remove()
    baseLayer = null
  }
  if (segmentLayers.length) {
    segmentLayers.forEach((l) => l.remove())
    segmentLayers = []
  }

  const baseLatlngs = props.coords.map(([lon, lat]) => [lat, lon])
  baseLayer = L.polyline(baseLatlngs, { weight: 4, color: '#888', opacity: 0.35 }).addTo(map)

  const baseBounds = baseLayer.getBounds()
  const fallbackView = () => {
    if (baseLatlngs.length === 1) {
      map.setView(baseLatlngs[0], 15)
    } else if (baseBounds.isValid()) {
      map.fitBounds(baseBounds, { padding: [20, 20] })
    }
  }

  if (props.segments?.length) {
    const bounds = []
    for (const seg of props.segments) {
      const latlngs = seg.coords.map(([lon, lat]) => [lat, lon])
      const l = L.polyline(latlngs, { weight: 5, color: seg.color || '#3366cc' }).addTo(map)
      segmentLayers.push(l)
      bounds.push(...latlngs)
    }
    if (bounds.length) {
      map.fitBounds(bounds, { padding: [20, 20] })
    } else {
      fallbackView()
    }
  } else {
    fallbackView()
  }

  map.invalidateSize()
}

onBeforeUnmount(() => map && map.remove())
</script>

<template>
  <div>
    <div
      ref="mapEl"
      style="height: 360px; width: 100%; border: 1px solid #eee; border-radius: 8px"
    ></div>
    <div v-if="legend?.length" style="display: flex; gap: 12px; flex-wrap: wrap; margin-top: 8px">
      <div v-for="item in legend" :key="item.label" style="display: flex; align-items: center; gap: 6px">
        <span :style="{ width: '16px', height: '4px', backgroundColor: item.color, display: 'inline-block' }"></span>
        <span style="font-size: 12px">{{ item.label }}</span>
      </div>
    </div>
  </div>
</template>
