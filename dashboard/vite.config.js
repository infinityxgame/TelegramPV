import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { fileURLToPath, URL } from 'node:url'
import fs from 'node:fs'
import path from 'node:path'
import os from 'node:os'

let backendPort = process.env.TELEGRAMPV_PORT || '8766'
let backendHost = process.env.TELEGRAMPV_BIND_HOST || '127.0.0.1'
if (backendHost === '0.0.0.0') {
  backendHost = '127.0.0.1'
}

try {
  const userEnvPath = path.join(os.homedir(), '.telegrampv', '.env')
  if (fs.existsSync(userEnvPath)) {
    const lines = fs.readFileSync(userEnvPath, 'utf8').split('\n')
    for (const rawLine of lines) {
      const line = rawLine.trim()
      if (line.startsWith('PORT=')) {
        const val = line.split('=')[1]?.trim()
        if (val) backendPort = val
      } else if (line.startsWith('BIND_HOST=')) {
        const val = line.split('=')[1]?.trim()
        if (val && val !== '0.0.0.0') backendHost = val
      }
    }
  }
} catch (e) {
  console.error(e)
}

const proxyTarget = `http://${backendHost}:${backendPort}`

export default defineConfig({
  plugins: [vue()],
  server: {
    port: 8082,
    strictPort: true,
    proxy: {
      '/api': {
        target: proxyTarget,
        changeOrigin: true
      }
    }
  },
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url))
    }
  },
  build: {
    outDir: 'dist',
    emptyOutDir: true
  }
})
