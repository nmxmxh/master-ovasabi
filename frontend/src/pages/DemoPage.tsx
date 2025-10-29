import React from 'react';
import styled from 'styled-components';
import { useCampaignData } from '../providers/CampaignProvider';
import serviceRegistration from '../../config/service_registration.json';
import EnhancedParticleSystem from '../components/EnhancedParticleSystem';

const DemoPageContainer = styled.div`
  display: flex;
  flex-direction: column;
  gap: 24px;
`;

const Section = styled.div`
  background: #111;
  border: 1px solid #333;
  padding: 16px;
`;

const SectionTitle = styled.h2`
  font-size: 16px;
  font-weight: bold;
  margin-bottom: 12px;
  color: #fff;
  text-transform: uppercase;
  letter-spacing: 1px;
`;

const ParticleContainer = styled.div`
  height: 500px;
  width: 100%;
  position: relative;
  background: #000;
  border: 1px solid #333;
`;

const CampaignList = styled.ul`
  list-style: none;
  padding: 0;
  margin: 0;
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(250px, 1fr));
  gap: 12px;
`;

const CampaignItem = styled.li`
  background: #000;
  border: 1px solid #333;
  padding: 12px;
  font-family: 'Monaco', 'Menlo', 'Consolas', monospace;
  font-size: 12px;
`;

const ServiceList = styled.div`
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  gap: 12px;
`;

const ServiceCard = styled.div`
  background: #000;
  border: 1px solid #333;
  padding: 12px;
`;

function CampaignsSection() {
  const { campaigns, loading, error } = useCampaignData();

  return (
    <Section>
      <SectionTitle>Live Campaigns</SectionTitle>
      {loading && <div className="minimal-text">Loading campaigns...</div>}
      {error && <div className="minimal-text">Error: {error}</div>}
      {!loading && !error && (
        <CampaignList>
          {campaigns.map(campaign => (
            <CampaignItem key={campaign.id}>
              <div className="minimal-card-title" style={{ fontSize: '13px' }}>
                {campaign.title || campaign.name}
              </div>
              <div className="minimal-card-id">ID: {campaign.id}</div>
              <span className={`minimal-status ${campaign.status || 'unknown'}`}>
                {campaign.status || 'UNKNOWN'}
              </span>
            </CampaignItem>
          ))}
        </CampaignList>
      )}
    </Section>
  );
}

function ServicesSection() {
  return (
    <Section>
      <SectionTitle>Backend Services</SectionTitle>
      <ServiceList>
        {serviceRegistration.map((service: any) => (
          <ServiceCard key={service.name}>
            <div className="minimal-card-title" style={{ fontSize: '13px' }}>
              {service.name}
            </div>
            <div className="minimal-text">Version: {service.version}</div>
            <div className="minimal-text">
              Capabilities: {service.capabilities?.join(', ') || 'N/A'}
            </div>
          </ServiceCard>
        ))}
      </ServiceList>
    </Section>
  );
}

function ThreeDeeSection() {
  return (
    <Section>
      <SectionTitle>3D Visualization</SectionTitle>
      <ParticleContainer>
        <EnhancedParticleSystem />
      </ParticleContainer>
    </Section>
  );
}

const DemoPage: React.FC = () => {
  return (
    <DemoPageContainer>
      <CampaignsSection />
      <ThreeDeeSection />
      <ServicesSection />
    </DemoPageContainer>
  );
};

export default DemoPage;
