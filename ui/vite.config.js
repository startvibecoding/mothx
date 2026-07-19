import { svelte } from '@sveltejs/vite-plugin-svelte';
import { defineConfig } from 'vite';

export default defineConfig({
  plugins: [svelte()],
  server: {
    proxy: {
      '/api': 'http://127.0.0.1:7872',
      '/v1': 'http://127.0.0.1:7872',
      '/health': 'http://127.0.0.1:7872',
      '/ws': {
        target: 'ws://127.0.0.1:7872',
        ws: true
      }
    }
  }
});
