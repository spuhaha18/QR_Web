import { defineConfig } from 'vite';
import { svelte } from '@sveltejs/vite-plugin-svelte';

// Build output → web/dist (Go embed target). base './' for embed-root serving.
export default defineConfig({
  plugins: [svelte()],
  base: './',
  build: {
    outDir: '../dist',
    emptyOutDir: true,
  },
  server: {
    port: 5173,
    proxy: {
      // dev: proxy backend API to Go server (Fiber on :5000)
      '/create_label': 'http://localhost:5000',
      '/api': 'http://localhost:5000',
    },
  },
});
