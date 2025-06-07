import { create } from "zustand";
import { persist } from "zustand/middleware";
import type { UIState } from "../types/ui-state";

export const useUIStore = create<UIState>()(
  persist(
    (set) => ({
      theme: window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light",
      direction: document?.dir === "rtl" ? "rtl" : "ltr",
      locale: navigator.language || "en-US",
      scriptCode: undefined,

      orientation: window.innerWidth > window.innerHeight ? "landscape" : "portrait",
      viewport: {
        width: window.innerWidth,
        height: window.innerHeight,
        dvw: window.innerWidth,
        dvh: window.innerHeight,
      },

      motion: {
        prefersReducedMotion: window.matchMedia("(prefers-reduced-motion: reduce)").matches,
      },

      contrast: {
        prefersHighContrast: window.matchMedia("(prefers-contrast: more)").matches,
        colorContrastRatio: undefined,
      },

      animationDirection: "forward",
      menu: "close",
      footerLink: "",

      setUIState: (partial: Partial<UIState>) => set((state) => ({ ...state, ...partial })),
    }),
    {
      name: "web-ovasabi",
    }
  )
);

export const getUIStore = () => useUIStore.getState();
