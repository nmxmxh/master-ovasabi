// Component registry for simple UI components
import {
  HeroComponent,
  EditorComponent,
  CardComponent,
  GridComponent,
  TextComponent,
  ButtonComponent,
  DefaultComponent
} from './SimpleComponents';

// Simple component registry - maps component types to React components
export const COMPONENT_REGISTRY = {
  hero: HeroComponent,
  editor: EditorComponent,
  card: CardComponent,
  grid: GridComponent,
  text: TextComponent,
  button: ButtonComponent,
  // Legacy support for existing types
  Box: CardComponent,
  Grid: GridComponent,
  Button: ButtonComponent,
  Text: TextComponent
};

// Simple component renderer
export const renderSimpleComponent = (componentName: string, component: any, theme: any = {}) => {
  const ComponentType =
    COMPONENT_REGISTRY[component.type as keyof typeof COMPONENT_REGISTRY] || DefaultComponent;

  console.log(`[SimpleRenderer] Rendering ${componentName}:`, {
    type: component.type,
    props: Object.keys(component),
    hasTheme: !!theme
  });

  return <ComponentType key={componentName} {...component} theme={theme} />;
};

// Helper to extract UI components from campaign data
export const extractUIComponents = (campaign: any) => {
  // Simple extraction - try multiple common locations
  const uiComponents =
    campaign?.ui_components ||
    campaign?.serviceSpecific?.campaign?.ui_components ||
    campaign?.metadata?.service_specific?.campaign?.ui_components ||
    {};

  console.log('[SimpleRenderer] Extracted UI components:', {
    componentCount: Object.keys(uiComponents).length,
    components: Object.keys(uiComponents),
    campaignId: campaign?.id || campaign?.campaignId
  });

  return uiComponents;
};

// Helper to extract theme from campaign data
export const extractTheme = (campaign: any) => {
  const theme =
    campaign?.theme ||
    campaign?.serviceSpecific?.campaign?.theme ||
    campaign?.metadata?.service_specific?.campaign?.theme ||
    {};

  console.log('[SimpleRenderer] Extracted theme:', {
    themeKeys: Object.keys(theme),
    hasPrimaryColor: !!theme.primary_color,
    hasBackgroundColor: !!theme.background_color
  });

  return theme;
};
