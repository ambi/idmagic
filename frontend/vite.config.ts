import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import { tanstackRouter } from '@tanstack/router-plugin/vite'

// The dev server port and API proxy target are overridable so isolated
// environments (e.g. E2E on 5174/8082) can run alongside `just dev` (5173/8081).
const devPort = Number(process.env.VITE_DEV_PORT ?? 5173)
const apiTarget = process.env.VITE_API_TARGET ?? 'http://localhost:8081'

export default defineConfig({
  plugins: [
    tanstackRouter({
      target: 'react',
      autoCodeSplitting: true,
      codeSplittingOptions: {
        defaultBehavior: [['loader', 'component']],
      },
    }),
    react(),
    tailwindcss(),
  ],
  base: '/',
  server: {
    port: devPort,
    strictPort: true,
    proxy: {
      '^/realms/[^/]+/(api|scim|saml|federationmetadata|wsfed|trust|authorize|token|revoke|introspect|userinfo|register|par|device_authorization|end_session|\\.well-known|jwks|tenant-branding-assets)(/|\\?|$)':
        apiTarget,
      '/api': apiTarget,
      '/scim': apiTarget,
      '/saml': apiTarget,
      '/federationmetadata': apiTarget,
      '/wsfed': apiTarget,
      '/trust': apiTarget,
      '/authorize': apiTarget,
      '/token': apiTarget,
      '/revoke': apiTarget,
      '/introspect': apiTarget,
      '/userinfo': apiTarget,
      '/register': apiTarget,
      '/par': apiTarget,
      '/device_authorization': apiTarget,
      '/end_session': apiTarget,
      '/.well-known': apiTarget,
      '/jwks': apiTarget,
      '/tenant-branding-assets': apiTarget,
      '/health': apiTarget,
    },
  },
  build: {
    cssCodeSplit: false,
  },
})
