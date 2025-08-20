import React from 'react';
import { createRoot } from 'react-dom/client';
import './index.css';
import App from './App.tsx';
import GlobalStyles from './styles/global.ts';
import './main.ts';
import { registerServiceWorker } from './registerServiceWorker';
registerServiceWorker();

createRoot(document.getElementById('root')!).render(
  <>
    <GlobalStyles />
    <App />
  </>
);
