'use client';

import { createGlobalStyle } from 'styled-components';
import { scrollBarLook } from './scroll-bar';

export const GlobalStyles = createGlobalStyle`
:root {
  --width: 85%;
  --max-percentage-width: 95%;
  --max-desktop-width: 1050px;
  font-synthesis: none;
  text-rendering: optimizeLegibility;
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
  -webkit-text-size-adjust: 100%;


  @media (prefers-reduced-motion: no-preference) {
    scroll-behavior: smooth;
  }
}

* {
  -webkit-tap-highlight-color: rgba(0,0,0,0);
}

/* Allow interactions on canvas elements for OrbitControls */
canvas {
  user-select: auto;
  -webkit-user-select: auto;
  -moz-user-select: auto;
  -ms-user-select: auto;
  -webkit-tap-highlight-color: initial;
  touch-action: none; /* Prevent default touch behaviors for better 3D controls */
  cursor: grab; /* Default cursor for 3D interaction */
}

canvas:active {
  cursor: grabbing; /* Cursor when dragging */
}

*,
*:before,
*:after {
  box-sizing: border-box;
  border: 0;
  padding: 0;
  margin: 0;
}

html,
body {
  width: 100%;
  margin: 0;
  padding: 0;
  outline: none;
  transform: translate3d(0, 0, 0);
  font-family: "Geist";
  
  scroll-behavior: smooth !important;
  /* https://github.com/vercel/next.js/issues/51721 */

  user-select: none;
  position: relative;

  ${scrollBarLook}
}

a,
button {
  color: inherit;
  cursor: pointer;
  text-decoration: none;
  outline: none;
  -webkit-touch-callout: none;
  /* Allow button interactions - don't disable user-select for buttons */
  -webkit-tap-highlight-color: rgba(0, 0, 0, 0.1); /* Subtle tap feedback */
  background-color: transparent;
  font-size: inherit;
  color: inherit;
  /* Ensure buttons are interactive */
  pointer-events: auto;
  touch-action: manipulation; /* Better touch response */
}

/* Ensure interactive elements in particle system UIs work properly */
button:hover {
  -webkit-tap-highlight-color: rgba(255, 255, 255, 0.1);
}

button:active {
  -webkit-tap-highlight-color: rgba(255, 255, 255, 0.2);
  transform: scale(0.98); /* Subtle press feedback */
}


html {
  --full-screen-w: 100svw;
  --full-screen-h: 100svh;
}

div {
  box-sizing: border-box;
}

@media (orientation: landscape) {
  html {
    --full-screen-w: 100dvw;
    --full-screen-h: 100dvh;
  }

  #root {
    flex-direction: row;
  }
}
`;

export default GlobalStyles;

//*  Documentation Source: https://www.cyishere.dev/blog/design-system-with-styled-components
