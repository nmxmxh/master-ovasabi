// types/ui.ts
export type FooterLink = "" | "about-us" | "projects" | "shop";

export interface UIState {
  theme: "light" | "dark";
  direction: "ltr" | "rtl";
  locale: string;
  scriptCode?: string; // Optional ISO 15924 script
  orientation: "portrait" | "landscape";
  viewport: {
    width: number;
    height: number;
    dvw: number;
    dvh: number;
  };
  motion: {
    prefersReducedMotion: boolean;
  };
  contrast: {
    prefersHighContrast: boolean;
    colorContrastRatio?: string;
  };
  animationDirection: "forward" | "backward";
  menu: "close" | "open";
  footerLink: FooterLink;
}
