// Campaign-related type definitions
export interface CampaignMetadata {
  campaignId: number | string;
  campaignName?: string;
  slug?: string;
  features: string[];
  // Core campaign fields
  title?: string;
  description?: string;
  status?: 'active' | 'inactive' | 'draft';
  tags?: string[];
  createdAt?: string;
  updatedAt?: string;
  // Backend campaign structure
  about?: {
    order?: Array<{
      p?: string;
      title?: string;
      type?: 'content' | 'list' | 'image' | 'video';
      list?: string[];
      subtitle?: string;
    }>;
  };
  ui_content?: {
    banner?: string;
    cta?: string;
    architecture_overview?: {
      description?: string;
      sections?: string[];
    };
    lead_form?: {
      fields?: string[];
      submit_text?: string;
    };
    [key: string]: any;
  };
  broadcast_enabled?: boolean;
  channels?: string[];
  i18n_keys?: string[];
  focus?: string;
  inos_enabled?: boolean;
  ranking_formula?: string;
  start_date?: string;
  end_date?: string;
  owner_id?: string;
  master_id?: number;
  master_uuid?: string;
  // Service-specific data
  serviceSpecific?: {
    campaign?: Record<string, any>;
    localization?: {
      scripts?: Record<string, ScriptBlock>;
      scripts_translations?: Record<string, any>;
      scripts_translated?: Record<string, ScriptBlock>;
    };
    [key: string]: any;
  };
  // Campaign switch tracking
  last_switched?: string;
  switch_reason?: string;
  switch_status?: string;
  scheduling?: Record<string, any>;
  versioning?: Record<string, any>;
  audit?: Record<string, any>;
  gdpr?: {
    consentRequired: boolean;
    privacyPolicyUrl?: string;
    termsUrl?: string;
    consentGiven?: boolean;
    consentTimestamp?: string; // ISO string with timezone
  };
}

export interface ScriptBlock {
  main_text: string;
  options_title: string;
  options_subtitle: string;
  question_subtitle: string;
  questions: Array<{
    question: string;
    why_this_matters: string;
    options: string[];
    accessibility?: {
      ariaLabel?: string;
      altText?: string;
    };
  }>;
}

export interface CampaignState {
  campaignState?: any; // Legacy field for campaign state integration
}
