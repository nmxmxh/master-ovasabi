import React from 'react';
import type { CampaignMetadata } from '../store/types/campaign';
import DummyComponentGenerator from './DummyComponentGenerator';

interface CampaignUIRendererProps {
  campaign?: CampaignMetadata;
  isLoading?: boolean;
}

const CampaignUIRenderer: React.FC<CampaignUIRendererProps> = ({ campaign, isLoading = false }) => {
  if (isLoading) {
    return <div className="minimal-text">Loading UI components...</div>;
  }

  if (!campaign) {
    return <div className="minimal-text">No campaign data provided.</div>;
  }

  const campaignData = (campaign as any).metadata?.service_specific?.campaign || campaign.serviceSpecific?.campaign || campaign;
  const uiComponents = campaignData.ui_components || (campaign as any).ui_components || {};
  const theme = campaignData.theme || (campaign as any).theme || {};

  return (
    <div className="minimal-section">
      <div className="minimal-title">UI Components</div>
      <DummyComponentGenerator components={uiComponents} theme={theme} />
    </div>
  );
};

export default CampaignUIRenderer;
