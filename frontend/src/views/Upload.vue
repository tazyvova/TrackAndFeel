<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'

const file = ref(null)
const uploading = ref(false)
const msg = ref('')
const router = useRouter()

async function onSubmit(e) {
  e.preventDefault()
  msg.value = ''
  if (!file.value?.files?.[0]) {
    msg.value = 'Pick a GPX file first.'
    return
  }
  const form = new FormData()
  form.append('file', file.value.files[0])

  uploading.value = true
  try {
    const res = await fetch('/api/upload', { method: 'POST', body: form })
    if (!res.ok) throw new Error(`Upload failed: HTTP ${res.status}`)
    const data = await res.json()
    msg.value = 'Uploaded. New ID: ${data.id}'
    // go view it
    router.push('/activities/${data.id}')
  } catch (err) {
    msg.value = String(err)
  } finally {
    uploading.value = false
  }
}
</script>

<template>
  <h2>Upload GPX</h2>
  <form @submit="onSubmit">
    <input ref="file" type="file" accept=".gpx" />
    <button :disabled="uploading" type="submit">
      {{ uploading ? 'Uploading…' : 'Upload' }}
    </button>
  </form>
  <p v-if="msg">{{ msg }}</p>
  <p style="color: #666; margin-top: 8px">
    Tip: you can drag and drop a <code>.gpx</code> exported from Garmin.
  </p>
</template>
