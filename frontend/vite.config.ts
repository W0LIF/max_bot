import react from '@vitejs/plugin-react'
import { defineConfig } from 'vite'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  server: {
    // В проде /api — это Vercel-функции из корневой api/.
    // Локально их эмулирует `go run ./cmd/server` (порт 8080), туда и проксируем.
    // Переопределяется переменной VITE_API_PROXY_TARGET.
    proxy: {
      '/api': {
        target: process.env.VITE_API_PROXY_TARGET || 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
})
