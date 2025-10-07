import React from 'react';
import type { Campaign } from '../store/types/campaign';
import DummyComponentGenerator from './DummyComponentGenerator';
import { extractUIComponents, extractTheme } from './ui/ComponentRegistry';

interface CampaignUIRendererProps {
  campaign?: Campaign;
  isLoading?: boolean;
}

const CampaignUIRenderer: React.FC<CampaignUIRendererProps> = ({ campaign, isLoading = false }) => {
  if (isLoading) {
    return <div className="minimal-text">Loading UI components...</div>;
  }

  if (!campaign) {
    return <div className="minimal-text">No campaign data provided.</div>;
  }

  const uiComponents = extractUIComponents(campaign);
  const theme = extractTheme(campaign);

  return (
    <div className="minimal-section">
      <div className="minimal-title">UI Components</div>
      <DummyComponentGenerator components={uiComponents} theme={theme} />
    </div>
  );
};

export default CampaignUIRenderer;