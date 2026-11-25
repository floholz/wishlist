import './lib/storage-migrate' // must run before any module reads localStorage
import { mount } from 'svelte'
import './app.css'
import { registerSW } from 'virtual:pwa-register'
import App from './App.svelte'

export default mount(App, { target: document.getElementById('app')! })

registerSW({ immediate: true })
