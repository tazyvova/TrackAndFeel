import { defineStore } from 'pinia'

export const useActivitiesStore = defineStore('activities', {
  state: () => ({
    items: [],
    totalKnown: 0,         // optional if you add count later
    loading: false,
    error: '',
    details: {},           // cache: { [id]: { summary, geojson, series } }
    unit: 'kmh',           // 'kmh' | 'mps' | 'pace' (min/km)
  }),
  actions: {
    async fetchList(limit = 20, offset = 0) {
      this.loading = true
      this.error = ''
      try {
        const res = await fetch(`/api/activities?limit=${limit}&offset=${offset}`)
        if (!res.ok) throw new Error(`HTTP ${res.status}`)
        const data = await res.json()
        this.items = data.items || []
      } catch (e) {
        this.error = String(e)
      } finally {
        this.loading = false
      }
    },
    async fetchDetail(id) {
      if (this.details[id]) return this.details[id]
      this.loading = true
      this.error = ''
      try {
        const res = await fetch(`/api/activities/${id}/track`)
        if (!res.ok) throw new Error(`HTTP ${res.status}`)
        const data = await res.json()
        this.details[id] = data
        return data
      } catch (e) {
        this.error = String(e)
        throw e
      } finally {
        this.loading = false
      }
    },
    setUnit(u) {
      this.unit = u
    }
  }
})
