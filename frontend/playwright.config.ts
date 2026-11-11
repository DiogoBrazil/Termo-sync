import {defineConfig} from '@playwright/test'
export default defineConfig({testDir:'./e2e',workers:1,use:{baseURL:'http://127.0.0.1:5173',headless:true,viewport:{width:1366,height:768},colorScheme:'light'},webServer:{command:'npm run dev -- --port 5173',url:'http://127.0.0.1:5173',reuseExistingServer:true},reporter:'list',outputDir:'../build/browser-results'})
