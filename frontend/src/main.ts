import {createApp} from 'vue'
import App from './App.vue'
import './generated.css'
import './settings-generated.css'
import './style.css'
if(import.meta.env.DEV && new URLSearchParams(location.search).has('preview')) {
 await import('./preview').then(m=>m.installPreview())
}
if(!window.wails?.Call){
 const runtime='/wails/runtime.js'
 try{
  const mod=await import(/* @vite-ignore */ runtime)
  if(mod?.Call?.ByName)window.wails={Call:mod.Call}
 }catch{ /* The UI explains that native operations require Windows. */ }
}
createApp(App).mount('#app')
