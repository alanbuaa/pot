/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{vue,js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        'bg-primary': '#0a0e27',
        'bg-secondary': '#141d3a',
        'border': '#1e3a5f',
        'committee-leader': '#ffd700',
        'committee-member': '#00ff88',
        'pot-mining': '#00d4ff',
        'pot-normal': '#4488ff',
        'transaction': '#ffa500',
        'incentive': '#ffd700',
        'storage': '#9370db',
        'vdf': '#ff69b4'
      }
    },
  },
  plugins: [],
}
