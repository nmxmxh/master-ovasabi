import { useMemo } from 'react';
import { useCampaignState } from '../store/hooks/useCampaign';

/**
 * Hook for rendering campaign-specific UI components
 * Leverages existing campaign state management
 */
export function useCampaignUI() {
  const { state: campaignState } = useCampaignState();

  // Extract UI content and theme from campaign state
  const uiContent = useMemo(() => {
    return campaignState.ui_content || {};
  }, [campaignState.ui_content]);

  const theme = useMemo(() => {
    return campaignState.serviceSpecific?.campaign?.theme || {};
  }, [campaignState.serviceSpecific]);

  const features = useMemo(() => {
    return campaignState.features || [];
  }, [campaignState.features]);

  const aboutContent = useMemo(() => {
    return campaignState.about || {};
  }, [campaignState.about]);

  // Check if specific features are enabled
  const hasFeature = useMemo(() => {
    return (feature: string) => features.includes(feature);
  }, [features]);

  // Get campaign-specific configuration
  const getConfig = useMemo(() => {
    return (key: string, defaultValue?: any) => {
      return uiContent[key] || defaultValue;
    };
  }, [uiContent]);

  // Render campaign-specific content
  const renderContent = useMemo(() => {
    return (contentKey: string) => {
      const content = uiContent[contentKey];
      if (!content) return null;

      // Handle different content types
      if (typeof content === 'string') {
        return content;
      }

      if (typeof content === 'object' && content !== null) {
        // Handle structured content like lead_form
        if (content.fields && Array.isArray(content.fields)) {
          return {
            type: 'form',
            fields: content.fields,
            submitText: content.submit_text || 'Submit'
          };
        }

        // Handle architecture overview
        if (content.description && content.sections) {
          return {
            type: 'overview',
            description: content.description,
            sections: content.sections
          };
        }

        return content;
      }

      return null;
    };
  }, [uiContent]);

  // Get about content in order
  const getAboutContent = useMemo(() => {
    return () => {
      if (!aboutContent.order || !Array.isArray(aboutContent.order)) {
        return [];
      }

      return aboutContent.order.map((item: any, index: number) => ({
        id: index,
        type: item.type || 'content',
        title: item.title,
        subtitle: item.subtitle,
        content: item.p,
        list: item.list || []
      }));
    };
  }, [aboutContent]);

  return {
    // Campaign data
    campaignState,
    uiContent,
    theme,
    features,
    aboutContent,

    // Utility functions
    hasFeature,
    getConfig,
    renderContent,
    getAboutContent,

    // Quick checks
    isActive: campaignState.status === 'active',
    hasUI: Object.keys(uiContent).length > 0,
    hasTheme: Object.keys(theme).length > 0,
    hasAbout: aboutContent.order && aboutContent.order.length > 0
  };
}

/**
 * Hook for campaign service integration
 * Based on the features array in campaign state
 */
export function useCampaignServices() {
  const { features, hasFeature } = useCampaignUI();

  // Service availability checks
  const services = useMemo(() => {
    return {
      messaging: hasFeature('messaging'),
      media: hasFeature('media'),
      notification: hasFeature('notification'),
      analytics: hasFeature('analytics'),
      search: hasFeature('search'),
      localization: hasFeature('localization'),
      contentmoderation: hasFeature('contentmoderation'),
      waitlist: hasFeature('waitlist'),
      referral: hasFeature('referral'),
      leaderboard: hasFeature('leaderboard'),
      broadcast: hasFeature('broadcast')
    };
  }, [hasFeature]);

  // Get active services count
  const activeServicesCount = useMemo(() => {
    return Object.values(services).filter(Boolean).length;
  }, [services]);

  return {
    services,
    activeServicesCount,
    features,
    hasFeature
  };
}
