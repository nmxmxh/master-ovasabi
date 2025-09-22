import React from 'react';
import type { CampaignMetadata } from '../store/types/campaign';
import { renderSimpleComponent, extractUIComponents, extractTheme } from './ui/ComponentRegistry';

interface SimpleCampaignUIRendererProps {
  campaign?: CampaignMetadata;
  isLoading?: boolean;
}

const SimpleCampaignUIRenderer: React.FC<SimpleCampaignUIRendererProps> = ({
  campaign,
  isLoading = false
}) => {
  console.log('[SimpleCampaignUIRenderer] Render called:', {
    campaign: campaign
      ? {
          id: campaign.campaignId || (campaign as any).id,
          title: campaign.title,
          status: campaign.status,
          hasUIComponents: !!(campaign as any).ui_components
        }
      : null,
    isLoading
  });

  if (isLoading) {
    return (
      <div
        style={{
          fontFamily: 'Monaco, Menlo, Consolas, monospace',
          background: '#000',
          color: '#fff',
          textAlign: 'center',
          padding: '40px',
          fontSize: '12px'
        }}
      >
        Loading campaign interface...
      </div>
    );
  }

  if (!campaign) {
    return (
      <div
        style={{
          fontFamily: 'Monaco, Menlo, Consolas, monospace',
          background: '#000',
          color: '#666',
          textAlign: 'center',
          padding: '40px',
          fontSize: '12px'
        }}
      >
        <div>No campaign selected</div>
        <div>Select a campaign to view its interface</div>
      </div>
    );
  }

  // Simple data extraction - single source of truth
  const uiComponents = extractUIComponents(campaign);
  const theme = extractTheme(campaign);

  const campaignTitle = campaign.title || 'Campaign Interface';
  const campaignDescription = campaign.description || 'Welcome to your campaign dashboard';

  console.log('[SimpleCampaignUIRenderer] Extracted data:', {
    uiComponents: Object.keys(uiComponents),
    theme: Object.keys(theme),
    campaignTitle,
    campaignDescription
  });

  // Apply theme styles to the main container
  const containerStyles = {
    fontFamily: theme.font_family || 'Monaco, Menlo, Consolas, monospace',
    background: theme.background_color || '#000',
    color: theme.text_color || '#fff',
    fontSize: '12px',
    lineHeight: '1.4',
    minHeight: '400px'
  };

  return (
    <div style={containerStyles}>
      {/* Campaign Header */}
      <div
        style={{
          padding: '20px',
          borderBottom: `1px solid ${theme.border_color || '#333'}`,
          background: theme.background_color || '#111',
          textAlign: 'center'
        }}
      >
        <h1
          style={{
            fontSize: '24px',
            fontWeight: 'bold',
            marginBottom: '8px',
            color: theme.primary_color || '#fff'
          }}
        >
          {campaignTitle}
        </h1>
        <p
          style={{
            fontSize: '14px',
            color: theme.text_color || '#ccc',
            margin: 0
          }}
        >
          {campaignDescription}
        </p>
      </div>

      {/* UI Components */}
      {Object.keys(uiComponents).length > 0 ? (
        <div style={{ padding: '20px' }}>
          {Object.entries(uiComponents).map(([componentName, component]) => (
            <div key={componentName} style={{ marginBottom: '20px' }}>
              <div
                style={{
                  fontSize: '12px',
                  fontWeight: 'bold',
                  color: theme.primary_color || '#0f0',
                  marginBottom: '8px',
                  textTransform: 'uppercase',
                  letterSpacing: '0.5px'
                }}
              >
                {componentName.replace(/_/g, ' ')}
              </div>
              {renderSimpleComponent(componentName, component, theme)}
            </div>
          ))}
        </div>
      ) : (
        /* Fallback when no UI components */
        <div
          style={{
            textAlign: 'center',
            padding: '60px 20px',
            background: theme.background_color || '#111',
            border: `1px solid ${theme.border_color || '#333'}`,
            margin: '20px',
            borderRadius: '8px'
          }}
        >
          <div
            style={{
              fontSize: '20px',
              fontWeight: 'bold',
              marginBottom: '12px',
              color: theme.primary_color || '#fff'
            }}
          >
            {campaignTitle}
          </div>
          <div
            style={{
              fontSize: '14px',
              color: theme.text_color || '#ccc',
              marginBottom: '24px',
              maxWidth: '600px',
              margin: '0 auto 24px auto',
              lineHeight: '1.5'
            }}
          >
            {campaignDescription}
          </div>
          <div
            style={{
              fontSize: '12px',
              color: theme.text_color || '#999',
              fontStyle: 'italic'
            }}
          >
            No UI components configured for this campaign
          </div>
        </div>
      )}

      {/* Campaign Info */}
      <div
        style={{
          padding: '20px',
          borderTop: `1px solid ${theme.border_color || '#333'}`,
          background: theme.background_color || '#111',
          fontSize: '11px',
          color: theme.text_color || '#999'
        }}
      >
        <div style={{ marginBottom: '8px' }}>
          <strong>Status:</strong> {campaign.status || 'Unknown'} |<strong> Platform:</strong>{' '}
          {(campaign as any).platform_type || (campaign as any).focus || 'general'} |
          <strong> Features:</strong> {campaign.features?.length || 0} enabled
        </div>
        <div>
          <strong>UI Components:</strong> {Object.keys(uiComponents).length} configured
        </div>
      </div>
    </div>
  );
};

export default SimpleCampaignUIRenderer;
