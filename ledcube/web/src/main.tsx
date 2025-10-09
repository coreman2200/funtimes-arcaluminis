import React from 'react'
import { createRoot } from 'react-dom/client'
import App from './ui/App'
import { ErrorBoundary } from './ui/ErrorBoundary'
import './styles.css';

const rootEl = document.getElementById('root')!
createRoot(rootEl).render(
  <ErrorBoundary>
    <App />
  </ErrorBoundary>
)