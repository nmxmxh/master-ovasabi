import React, { useState } from 'react';
import { getRequiredFields, validateFields } from '../utils/validateFields';
import { useCampaignOperations } from '../store/hooks/useCampaign';
import FormField from '../components/forms/FormField';
import InputField from '../components/forms/InputField';
import SelectField from '../components/forms/SelectField';
import TextareaField from '../components/forms/TextareaField';

const dummyCampaign = {
  title: 'My Awesome Campaign',
  description:
    'This is a default description for a new campaign. It outlines the main goals and objectives.',
  slug: 'my-awesome-campaign',
  tags: 'gaming, streaming, community',
  status: 'draft',
  focus: 'e-commerce'
};

function CreateCampaignPage() {
  const [campaign, setCampaign] = useState(dummyCampaign);
  const [status, setStatus] = useState('');
  const { createCampaign } = useCampaignOperations();

  const handleChange = (
    e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement>
  ) => {
    const { id, value } = e.target;
    setCampaign(prev => ({ ...prev, [id]: value }));
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!campaign.title.trim() || !campaign.slug.trim()) {
      alert('Title and Slug are required');
      return;
    }

    setStatus('Creating...');
    const payload = {
      ...campaign,
      metadata: {
        tags: campaign.tags
          .split(',')
          .map(tag => tag.trim())
          .filter(Boolean),
        focus: campaign.focus
      }
    };
    const requiredFields = getRequiredFields('campaign', 'create_campaign');
    const missingFields = validateFields(payload, requiredFields);
    if (missingFields.length > 0) {
      setStatus(`Missing required fields: ${missingFields.join(', ')}`);
      return;
    }
    try {
      const result: any = await createCampaign(payload);
      setStatus(`Success! Campaign created with ID: ${result.id || result.campaignId}`);
      alert('Campaign created successfully!');
      setCampaign(dummyCampaign); // Reset form to dummy data
    } catch (error: any) {
      console.error('Failed to create campaign:', error);
      const errorMessage = error?.message || (typeof error === 'string' ? error : 'Unknown error');
      setStatus(`Error: ${errorMessage}`);
      alert(`Failed to create campaign: ${errorMessage}`);
    }
  };

  return (
    <form onSubmit={handleSubmit}>
      <div className="minimal-section">
        <div className="minimal-title">CREATE NEW CAMPAIGN</div>
        <p className="minimal-text" style={{ marginBottom: '20px' }}>
          Use this form to define and launch a new campaign. Dummy data is pre-filled for your
          convenience.
        </p>

        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '24px' }}>
          <FormField label="Campaign Title" htmlFor="title">
            <InputField
              id="title"
              type="text"
              value={campaign.title}
              onChange={handleChange}
              required
            />
          </FormField>

          <FormField label="Slug" htmlFor="slug">
            <InputField
              id="slug"
              type="text"
              value={campaign.slug}
              onChange={handleChange}
              required
            />
          </FormField>

          <FormField label="Status" htmlFor="status">
            <SelectField id="status" value={campaign.status} onChange={handleChange}>
              <option value="draft">Draft</option>
              <option value="active">Active</option>
              <option value="archived">Archived</option>
            </SelectField>
          </FormField>

          <FormField label="Focus" htmlFor="focus">
            <InputField
              id="focus"
              type="text"
              value={campaign.focus}
              onChange={handleChange}
              placeholder="e.g., gaming, streaming, e-commerce"
            />
          </FormField>
        </div>

        <FormField label="Tags (comma-separated)" htmlFor="tags">
          <InputField id="tags" type="text" value={campaign.tags} onChange={handleChange} />
        </FormField>

        <FormField label="Description" htmlFor="description">
          <TextareaField
            id="description"
            value={campaign.description}
            onChange={handleChange}
            rows={5}
          />
        </FormField>

        <div
          style={{
            marginTop: '24px',
            borderTop: '1px solid #333',
            paddingTop: '16px',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between'
          }}
        >
          <button
            type="submit"
            className="minimal-button"
            disabled={status.includes('Creating...')}
          >
            {status.includes('Creating...') ? 'CREATING...' : 'Create Campaign'}
          </button>
          {status && (
            <div className="minimal-text" style={{ opacity: 0.8 }}>
              {status}
            </div>
          )}
        </div>
      </div>
    </form>
  );
}

export default CreateCampaignPage;
