import React from 'react';
import { useParams } from 'react-router-dom';
import { useCampaignState } from '../store/hooks/useCampaign';
import {
  extractViews,
  extractTheme,
  renderSimpleComponent
} from '../components/ui/ComponentRegistry';

const ViewPage: React.FC = () => {
  const { viewName } = useParams();
  const { state: campaign } = useCampaignState();

  const views = extractViews(campaign);
  const theme = extractTheme(campaign);
  const activeViewKey = viewName || 'main';
  const activeView = views[activeViewKey];

  if (!activeView) {
    return (
      <div className="minimal-section">
        <div className="minimal-title">VIEW NOT FOUND</div>
        <div className="minimal-text">No view named "{activeViewKey}"</div>
      </div>
    );
  }

  const components = activeView.components || {};

  return (
    <div className="minimal-section" style={{ background: '#000' }}>
      <div className="minimal-title">VIEW: {activeViewKey.toUpperCase()}</div>
      <div style={{ padding: '12px' }}>
        {Object.entries(components).map(([name, def]) => (
          <div key={name} style={{ marginBottom: '16px' }}>
            {renderSimpleComponent(name, def, theme)}
          </div>
        ))}
      </div>
    </div>
  );
};

export default ViewPage;
