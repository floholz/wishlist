import { defineConfig } from 'vite'
import { svelte } from '@sveltejs/vite-plugin-svelte'
import { VitePWA } from 'vite-plugin-pwa'

export default defineConfig({
  plugins: [
    svelte(),
    VitePWA({
      registerType: 'autoUpdate',
      includeAssets: ['favicon.svg', 'apple-touch-icon.png', 'presets/*'],
      manifest: {
        name: 'wishlist',
        short_name: 'wishlist',
        description: 'Simple wishlists. One link to edit, one link to share. No account needed.',
        lang: 'en',
        start_url: '/app',
        scope: '/',
        display: 'standalone',
        background_color: '#faf8f5',
        theme_color: '#faf8f5',
        icons: [
          { src: 'pwa-192.png', sizes: '192x192', type: 'image/png' },
          { src: 'pwa-512.png', sizes: '512x512', type: 'image/png' },
          { src: 'pwa-maskable-512.png', sizes: '512x512', type: 'image/png', purpose: 'maskable' },
        ],
      },
      workbox: {
        // precache the built UI only; API, files and the PocketBase admin always go to the network
        globPatterns: ['**/*.{js,css,html,svg,png,jpg,jpeg,webp,woff2}'],
        maximumFileSizeToCacheInBytes: 4 * 1024 * 1024,
        navigateFallback: 'index.html',
        navigateFallbackDenylist: [/^\/api\//, /^\/_\//],
        runtimeCaching: [],
      },
    }),
  ],
  server: {
    // during `npm run dev` proxy the API to the Go binary (`make dev`)
    proxy: { '/api': 'http://127.0.0.1:8080', '/_': 'http://127.0.0.1:8080' },
  },
})
