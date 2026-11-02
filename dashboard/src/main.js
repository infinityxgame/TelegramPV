import { createApp } from 'vue'
import App from './App.vue'
import './style.css'
import './themes-dl.css'
import { initTheme } from './utils/theme.js'

initTheme()

createApp(App).mount('#app')
