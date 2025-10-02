import React from 'react';
import { useParams, Link } from 'react-router-dom';
import { useCampaignData } from '../providers/CampaignProvider';
import CampaignUIRenderer from '../components/CampaignUIRenderer';

const DetailItem: React.FC<{ label: string; value?: string | string[] }> = ({ label, value }) => {
  if (!value) return null;
  return (
    <div>
      <div style={{ fontSize: '10px', color: '#888', textTransform: 'uppercase', marginBottom: '4px' }}>{label}</div>
      <div className="minimal-text" style={{ background: '#000', padding: '8px', border: '1px solid #333', minHeight: '20px' }}>
        {Array.isArray(value) ? value.join(', ') : value}
      </div>
    </div>
  );
};

const CampaignPage: React.FC = () => {
  const { slug } = useParams<{ slug: string }>();
  const { campaigns, loading } = useCampaignData();

  const campaign = campaigns.find(c => c.slug === slug);

  if (loading) {
    return <div className="minimal-text">Loading campaign...</div>;
  }

  if (!campaign) {
    return (
      <div className="minimal-section">
        <div className="minimal-title">Campaign Not Found</div>
        <p className="minimal-text">The campaign with slug "{slug}" could not be found.</p>
        <Link to="/" className="minimal-link" style={{ marginTop: '12px' }}>Back to Campaigns</Link>
      </div>
    );
  }

  const campaignData = (campaign as any).metadata?.service_specific?.campaign || campaign.serviceSpecific?.campaign || campaign;

  return (
    <div>
      <div className="minimal-section">
        <div className="minimal-title" style={{ fontSize: '18px' }}>{campaign.title}</div>
        <p className="minimal-text" style={{ marginBottom: '20px' }}>
          {campaign.description}
        </p>
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(250px, 1fr))', gap: '16px' }}>
          <DetailItem label="Slug" value={campaign.slug} />
          <DetailItem label="Status" value={campaign.status} />
          <DetailItem label="Focus" value={campaignData.focus} />
          <DetailItem label="Tags" value={campaign.tags} />
        </div>
      </div>

      <CampaignUIRenderer campaign={campaign} isLoading={loading} />
    </div>
  );
};

export default CampaignPage;
