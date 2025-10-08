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

// Minimal theme defaults to match site minimal styles
const minimalThemeDefaults = {
  primary_color: '#ffffff',
  secondary_color: '#333333',
  background_color: '#000000',
  text_color: '#ffffff',
  border_color: '#333333',
  font_family: 'Monaco, Menlo, Consolas, monospace'
};

// Normalize various ui_content shapes into renderable minimal components
const normalizeUIContent = (uiContent: any): Record<string, any> => {
  const normalized: Record<string, any> = {};
  if (!uiContent || typeof uiContent !== 'object') return normalized;

  Object.entries(uiContent).forEach(([key, value]) => {
    if (!value) return;

    // Direct hero sections
    if (key.toLowerCase().includes('hero') && typeof value === 'object') {
      normalized[key] = {
        type: 'hero',
        ...value
      };
      return;
    }

    // Textual blocks
    if (typeof value === 'string') {
      normalized[key] = { type: 'text', content: value };
      return;
    }

    // Generic objects with title/excerpt turn into cards
    if (typeof value === 'object') {
      const v: any = value;
      if (v.type && COMPONENT_REGISTRY[v.type as string as keyof typeof COMPONENT_REGISTRY]) {
        normalized[key] = v; // Already a known type
      } else if (v.title || v.excerpt || v.description) {
        normalized[key] = {
          type: 'card',
          title: v.title || key.replace(/_/g, ' '),
          excerpt: v.excerpt || v.description || '',
          author: v.author,
          claps: v.claps
        };
      } else {
        // Fallback to text dump of object
        normalized[key] = { type: 'text', content: JSON.stringify(v) };
      }
    }
  });

  return normalized;
};

// Helper to extract UI components from campaign data
export const extractUIComponents = (campaign: any) => {
  // Simple extraction - try multiple common locations
  const uiComponentsRaw =
    campaign?.ui_components ||
    campaign?.serviceSpecific?.campaign?.ui_components ||
    campaign?.metadata?.service_specific?.campaign?.ui_components ||
    {};

  const uiContentRaw =
    campaign?.ui_content ||
    campaign?.serviceSpecific?.campaign?.ui_content ||
    campaign?.metadata?.service_specific?.campaign?.ui_content ||
    {};

  // Merge explicit components with normalized ui_content
  const uiComponents = { ...normalizeUIContent(uiContentRaw), ...uiComponentsRaw };

  console.log('[SimpleRenderer] Extracted UI components:', {
    componentCount: Object.keys(uiComponents).length,
    components: Object.keys(uiComponents),
    campaignId: campaign?.id || campaign?.campaignId
  });

  return uiComponents;
};

// Helper to extract Views from campaign data
// views schema: { [viewName]: { components: { [key]: componentDef } } }
export const extractViews = (
  campaign: any
): Record<string, { components: Record<string, any> }> => {
  const viewsRaw =
    campaign?.views ||
    campaign?.serviceSpecific?.campaign?.views ||
    campaign?.metadata?.service_specific?.campaign?.views ||
    undefined;

  if (viewsRaw && typeof viewsRaw === 'object') {
    return viewsRaw as Record<string, { components: Record<string, any> }>;
  }

  // Back-compat: synthesize a default "main" view from ui_content/ui_components
  const components = extractUIComponents(campaign);
  return {
    main: { components }
  };
};

// Helper to extract theme from campaign data
export const extractTheme = (campaign: any) => {
  const providedTheme =
    campaign?.theme ||
    campaign?.serviceSpecific?.campaign?.theme ||
    campaign?.metadata?.service_specific?.campaign?.theme ||
    {};

  // Shallow merge minimal defaults with provided theme
  const theme = { ...minimalThemeDefaults, ...providedTheme };

  console.log('[SimpleRenderer] Extracted theme:', {
    themeKeys: Object.keys(theme),
    hasPrimaryColor: !!theme.primary_color,
    hasBackgroundColor: !!theme.background_color
  });

  return theme;
};
