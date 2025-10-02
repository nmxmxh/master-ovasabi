// Utility to validate required fields for any service/action using service_registration.json
import serviceRegistration from '../../config/service_registration.json';

export function getRequiredFields(service: string, action: string): string[] {
  const serviceObj = serviceRegistration.find((s: any) => s.name === service);
  if (!serviceObj || !serviceObj.action_map) return [];
  const actionMap = serviceObj.action_map as Record<string, any>;
  if (!actionMap[action]) return [];
  return actionMap[action].rest_required_fields || [];
}

export function validateFields(payload: any, requiredFields: string[]): string[] {
  return requiredFields.filter(field => {
    // Support nested fields like metadata.tag
    if (field.includes('.')) {
      const parts = field.split('.');
      let value = payload;
      for (const part of parts) {
        value = value?.[part];
        if (value === undefined || value === null) return true;
      }
      return false;
    }
    return payload[field] === undefined || payload[field] === null;
  });
}
