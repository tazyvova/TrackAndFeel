<script setup lang="ts">
async function ping() {
  const res = await fetch('/healthz')
  const txt = await res.text()
  alert('Backend says: ' + txt)
}
</script>

<template>
  <main style="max-width: 900px; margin: 40px auto; font-family: system-ui, sans-serif;">
    <h1>Training Insights</h1>
    <p>This is the Vue + Vite starter wired to a Go backend.</p>
    <button @click="ping">Check backend /healthz</button>

    <h2 style="margin-top: 24px;">Activities</h2>
    <pre style="background:#f6f6f6;padding:12px;border-radius:8px;">
GET /api/activities → []
    </pre>
  </main>
</template>

<style scoped>
button {
  padding: 8px 14px;
  border-radius: 8px;
  border: none;
  cursor: pointer;
}
button:hover { filter: brightness(0.95); }
</style>
