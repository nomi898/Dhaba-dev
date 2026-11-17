import { defineConfig, loadEnv } from 'vite';
import react from '@vitejs/plugin-react';

// Dev server proxies /api/* to the backend services, mirroring the production Nginx config.
// Targets come from env vars so nothing is hard-coded in application code.
export default defineConfig(({ mode }) => {
  const env = { ...process.env, ...loadEnv(mode, process.cwd(), '') };
  const target = (name, fallback) => env[name] || fallback;
  return {
    plugins: [react()],
    server: {
      port: 5173,
      proxy: {
        '/api/menu': target('MENU_API_URL', 'http://localhost:8000'),
        '/api/orders': target('ORDERS_API_URL', 'http://localhost:8081'),
        '/api/stats': target('STATS_API_URL', 'http://localhost:8082'),
      },
    },
    test: {
      environment: 'jsdom',
      globals: true,
      setupFiles: './src/test/setup.js',
      css: false,
    },
  };
});
