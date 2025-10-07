// Campaign-related type definitions
export interface Campaign {
  id: string;
  name: string;
  title: string;
  slug: string;
  description: string;
  status: 'active' | 'inactive' | 'draft';
  features: string[];
  tags: string[];
  createdAt?: string;
  updatedAt?: string;
  about?: Record<string, any>;
  ui_content?: Record<string, any>;
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
  serviceSpecific?: Record<string, any>;
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
    consentTimestamp?: string;
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