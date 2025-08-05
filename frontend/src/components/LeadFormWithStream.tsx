import React, { useState } from 'react';
import { useGlobalStore, useEventHistory, useMetadata } from '../store/global';

export const LeadFormWithStream: React.FC = () => {
  // Get live campaign state from Zustand
  const campaignState = useGlobalStore(state => state.campaignState) || {};
  // Fallback to empty object if not available
  const leadFormConfig = campaignState?.metadata?.service_specific?.ui_content?.lead_form || {};
  const fields: string[] = leadFormConfig.fields || ['name', 'email', 'referral_code'];
  const submitText: string = leadFormConfig.submit_text || 'Submit';

  const [form, setForm] = useState<Record<string, string>>({});
  const [loading, setLoading] = useState(false);
  const [success, setSuccess] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  const globalStore = useGlobalStore();
  const { metadata } = useMetadata();
  const events = useEventHistory(undefined, 10); // Show last 10 events

  // Handle input changes
  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setForm({ ...form, [e.target.name]: e.target.value });
  };

  // Handle form submission
  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setError(null);
    setSuccess(null);

    try {
      // Emit waitlist entry creation event (canonical)
      globalStore.emitEvent({
        type: 'waitlist:create:v1:requested',
        payload: {
          ...form,
          campaign_id: campaignState.slug || campaignState.campaignId || ''
        },
        metadata: {
          ...metadata,
          campaign: {
            ...metadata.campaign,
            slug: campaignState.slug || ''
          }
        }
      });

      // Emit broadcast event (canonical)
      globalStore.emitEvent({
        type: 'notification:broadcast:v1:requested',
        payload: {
          subject: 'New Lead',
          message: `New lead: ${form.name} (${form.email})`,
          campaign_id: campaignState.slug || campaignState.campaignId || '',
          ...form
        },
        metadata: {
          ...metadata,
          campaign: {
            ...metadata.campaign,
            slug: campaignState.slug || ''
          }
        }
      });

      setSuccess('Lead submitted and broadcasted!');
      setForm({});
    } catch (err: any) {
      setError('Failed to submit lead. Please try again.');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div style={{ maxWidth: 500, margin: '0 auto', padding: 24 }}>
      <h2>{campaignState.title || 'Lead Form'}</h2>
      <p>{campaignState.description || ''}</p>
      <form onSubmit={handleSubmit} style={{ marginBottom: 24 }}>
        {fields.map(field => (
          <div key={field} style={{ marginBottom: 12 }}>
            <label style={{ display: 'block', fontWeight: 500 }}>{field}</label>
            <input
              name={field}
              value={form[field] || ''}
              onChange={handleChange}
              required={field === 'name' || field === 'email'}
              style={{ width: '100%', padding: 8, borderRadius: 4, border: '1px solid #ccc' }}
            />
          </div>
        ))}
        <button type="submit" disabled={loading} style={{ padding: '10px 20px', fontWeight: 600 }}>
          {loading ? 'Submitting...' : submitText}
        </button>
        {success && <div style={{ color: 'green', marginTop: 12 }}>{success}</div>}
        {error && <div style={{ color: 'red', marginTop: 12 }}>{error}</div>}
      </form>

      <h3>Live Metadata</h3>
      <pre style={{ background: '#fafafa', padding: 12, borderRadius: 6, fontSize: 13 }}>
        {JSON.stringify(metadata, null, 2)}
      </pre>

      <h3>Recent Event Stream</h3>
      <ul style={{ background: '#f5f5f5', padding: 12, borderRadius: 6, fontSize: 13 }}>
        {events.map((event, idx) => (
          <li key={idx} style={{ marginBottom: 8 }}>
            <strong>{event.type}</strong> <br />
            <span style={{ color: '#888' }}>{event.timestamp}</span>
            <pre style={{ margin: 0, fontSize: 12 }}>{JSON.stringify(event.payload, null, 2)}</pre>
          </li>
        ))}
      </ul>
    </div>
  );
};
