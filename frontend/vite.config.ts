import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  server: {
    port: 5173,
    proxy: {
      '/api': 'http://backend:8080',
      '/healthz': 'http://backend:8080',
    },
  },
  preview: { port: 5173 },
})
