import { useEffect, useState } from 'react';

export interface BatteryInfo {
  charging: boolean;
  level: number;
  chargingTime: number;
  dischargingTime: number;
}

export function useBattery(): BatteryInfo | null {
  const [battery, setBattery] = useState<BatteryInfo | null>(null);

  useEffect(() => {
    let batteryManager: any;
    if ('getBattery' in navigator) {
      (navigator as any).getBattery().then((b: any) => {
        batteryManager = b;
        const update = () => {
          setBattery({
            charging: b.charging,
            level: b.level,
            chargingTime: b.chargingTime,
            dischargingTime: b.dischargingTime
          });
        };
        update();
        b.addEventListener('chargingchange', update);
        b.addEventListener('levelchange', update);
        b.addEventListener('chargingtimechange', update);
        b.addEventListener('dischargingtimechange', update);
      });
    }
    return () => {
      if (batteryManager) {
        batteryManager.removeEventListener('chargingchange', () => {});
        batteryManager.removeEventListener('levelchange', () => {});
        batteryManager.removeEventListener('chargingtimechange', () => {});
        batteryManager.removeEventListener('dischargingtimechange', () => {});
      }
    };
  }, []);

  return battery;
}
