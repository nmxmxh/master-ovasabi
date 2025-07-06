import { DateTime, Duration, Interval, Settings } from 'luxon';
import ianaTzData from 'iana-tz-data/iana-tz-data.json';

// --- useDate hook ---
// Provides timezone-aware, localized date/time utilities for UI
// Usage: const date = useDate(); date.format(...)

export interface UseDate {
  now: () => DateTime;
  fromISO: (iso: string, opts?: { zone?: string }) => DateTime;
  fromMillis: (ms: number, opts?: { zone?: string }) => DateTime;
  format: (dt: DateTime, opts?: { format?: string; locale?: string; zone?: string }) => string;
  toRelative: (dt: DateTime, base?: DateTime) => string | null;
  toLocale: (
    dt: DateTime,
    opts?: { locale?: string; zone?: string; options?: Intl.DateTimeFormatOptions }
  ) => string;
  duration: (from: DateTime, to: DateTime) => Duration;
  interval: (from: DateTime, to: DateTime) => Interval;
  listTimezones: () => string[];
  setDefaultZone: (zone: string) => void;
  getDefaultZone: () => string;
}

export function useDate(userZone?: string, userLocale?: string): UseDate {
  // Use browser or user-specified zone/locale
  const zone = userZone || Intl.DateTimeFormat().resolvedOptions().timeZone;
  const locale = userLocale || navigator.language || 'en-US';

  // Set Luxon defaults
  Settings.defaultZone = zone;
  Settings.defaultLocale = locale;

  return {
    now: () => DateTime.now().setZone(zone),
    fromISO: (iso, opts) => DateTime.fromISO(iso, { zone: opts?.zone || zone }),
    fromMillis: (ms, opts) => DateTime.fromMillis(ms, { zone: opts?.zone || zone }),
    format: (dt, opts) =>
      dt
        .setZone(opts?.zone || zone)
        .setLocale(opts?.locale || locale)
        .toFormat(opts?.format || 'FFF'),
    toRelative: (dt, base) => dt.setZone(zone).toRelative({ base: base || DateTime.now() }),
    toLocale: (dt, opts) =>
      dt
        .setZone(opts?.zone || zone)
        .setLocale(opts?.locale || locale)
        .toLocaleString(opts?.options || DateTime.DATETIME_MED),
    duration: (from, to) => to.diff(from),
    interval: (from, to) => Interval.fromDateTimes(from, to),
    listTimezones: () => {
      // ianaTzData.zoneData is an object: { [region]: { [city]: ... } }
      const zones: string[] = [];
      if (ianaTzData && typeof ianaTzData === 'object' && 'zoneData' in ianaTzData) {
        for (const region of Object.keys((ianaTzData as any).zoneData)) {
          for (const city of Object.keys((ianaTzData as any).zoneData[region])) {
            zones.push(`${region}/${city}`);
          }
        }
      }
      return zones;
    },
    setDefaultZone: z => (Settings.defaultZone = z),
    getDefaultZone: () => Settings.defaultZone.name
  };
}

// Example usage in a component:
// const date = useDate();
// const now = date.now();
// const formatted = date.format(now, { format: 'yyyy LLL dd, HH:mm ZZZZ' });
// const rel = date.toRelative(now.minus({ days: 1 }));
// const tzs = date.listTimezones();
