import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  build: {
    // 前端构建产物直接输出到 cmd/github-proxy/public，
    // 供 Go 的 //go:embed public/* 打包进二进制（见 cmd/github-proxy/main.go）
    outDir: '../cmd/github-proxy/public',
    emptyOutDir: true
  }
})
