import React from 'react';
import { createRoot } from 'react-dom/client';
import { ChakraProvider, createSystem, defaultConfig } from '@chakra-ui/react';
import './index.css';
import App from './App.tsx';
import GlobalStyles from './styles/global.ts';
import './main.ts';

// Custom Chakra UI theme using v3 API
const system = createSystem(defaultConfig, {
  theme: {
    tokens: {
      colors: {
        brand: {
          50: { value: '#e6f3ff' },
          100: { value: '#b3d9ff' },
          200: { value: '#80bfff' },
          300: { value: '#4da6ff' },
          400: { value: '#1a8cff' },
          500: { value: '#007bff' },
          600: { value: '#0066cc' },
          700: { value: '#005299' },
          800: { value: '#003d66' },
          900: { value: '#002833' }
        }
      }
    },
    semanticTokens: {
      colors: {
        bg: {
          value: { base: 'white', _dark: 'gray.900' }
        },
        text: {
          value: { base: 'gray.900', _dark: 'white' }
        }
      }
    }
  }
});

createRoot(document.getElementById('root')!).render(
  <ChakraProvider value={system}>
    <GlobalStyles />
    <App />
  </ChakraProvider>
);
