import React from 'react'
import {createRoot} from 'react-dom/client'
import './style.css'
import App from './App'
import {applyPlatformClass} from './utils/theme'

const container = document.getElementById('root')
if (!container) {
  throw new Error('Root element not found')
}

const root = createRoot(container)

void applyPlatformClass().finally(() => {
  root.render(
    <React.StrictMode>
      <App/>
    </React.StrictMode>
  )
})
