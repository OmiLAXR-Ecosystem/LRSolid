import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// https://vite.dev/config/
const API_HOST = process.env.API_HOST ?? 'http://localhost:8080'

export default defineConfig({
  plugins: [react()],
  server: {
    host: '0.0.0.0',
    port: 5173,
    proxy: {
      '/api':  API_HOST,
      '/xapi': API_HOST,
      '/docs': API_HOST,
    },
  },
})
