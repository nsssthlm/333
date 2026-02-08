export default defineNuxtConfig({
  devtools: { enabled: true },

  ssr: false,

  runtimeConfig: {
    public: {
      apiBaseUrl: process.env.VALVX_WEB_API_BASE_URL || 'https://api.valvx.se',
      speckleBaseUrl: process.env.VALVX_WEB_SPECKLE_BASE_URL || 'https://speckle.valvx.se',
      appOrigin: process.env.VALVX_WEB_SERVER_ORIGIN || 'https://app.valvx.se',
    },
  },

  app: {
    head: {
      title: 'ValvX',
      meta: [
        { charset: 'utf-8' },
        { name: 'viewport', content: 'width=device-width, initial-scale=1' },
      ],
    },
  },

  css: ['~/assets/css/main.css'],

  vite: {
    optimizeDeps: {
      include: ['three'],
    },
  },

  compatibilityDate: '2025-01-01',
})
