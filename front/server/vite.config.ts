import { fileURLToPath, URL } from 'node:url'
import vue from '@vitejs/plugin-vue'
import pxToRem from 'postcss-pxtorem'
import { defineConfig } from 'vitest/config'

export default defineConfig({
  plugins: [vue()],
  build: {
    // The admin SPA keeps its /manage/** routes while its hashed assets use a
    // dedicated prefix at the shared production gateway.
    assetsDir: 'manage-assets',
  },
  server: {
    proxy: {
      '/api': {
        target: 'http://127.0.0.1:8080',
        changeOrigin: true,
      },
    },
  },
  css: {
    postcss: {
      plugins: [
        pxToRem({
          rootValue: 100,
          propList: ['*'],
          unitPrecision: 5,
          replace: true,
          mediaQuery: false,
          minPixelValue: 2,
          exclude: /node_modules/i,
          selectorBlackList: ['.no-rem'],
        }),
      ],
    },
  },
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
  test: {
    environment: 'jsdom',
    setupFiles: ['./tests/unit/setup.ts'],
    exclude: ['tests/e2e/**', 'node_modules/**', 'dist/**'],
    restoreMocks: true,
  },
})
