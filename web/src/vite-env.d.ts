/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_API_BASE_URL: string
  readonly VITE_WS_URL: string
  readonly VITE_USE_MOCK: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}

// 声明图片类型
declare module '*.png'
declare module '*.jpg'
declare module '*.jpeg'
declare module '*.svg'
declare module '*.gif'