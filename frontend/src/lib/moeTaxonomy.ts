// Canonical MOE key vocabulary, mirroring docs/moe-taxonomy.md. Engines are
// free to emit keys not listed here — ResultsPanel falls back to a
// prettified raw key for anything unrecognized, so this is purely a display
// layer, not a schema.
export type MOECategory = 'attrition' | 'effectiveness' | 'sensor' | 'logistics';

export const MOE_CATEGORY_LABEL: Record<MOECategory, string> = {
  attrition: 'Attrition',
  effectiveness: 'Effectiveness',
  sensor: 'Sensor',
  logistics: 'Logistics',
};

interface MOEDef {
  label: string;
  category: MOECategory;
}

export const MOE_TAXONOMY: Record<string, MOEDef> = {
  blue_losses: { label: 'Blue losses', category: 'attrition' },
  red_losses: { label: 'Red losses', category: 'attrition' },
  avg_health_pct: { label: 'Avg. health remaining', category: 'attrition' },
  blue_kills: { label: 'Blue kills', category: 'effectiveness' },
  red_kills: { label: 'Red kills', category: 'effectiveness' },
  total_kills: { label: 'Total kills', category: 'effectiveness' },
  detections_total: { label: 'Total detections', category: 'sensor' },
  rounds_expended: { label: 'Rounds expended', category: 'logistics' },
};

const FALLBACK_CATEGORY: MOECategory = 'effectiveness';

export function moeLabel(key: string): string {
  return MOE_TAXONOMY[key]?.label ?? key.replace(/_/g, ' ');
}

export function moeCategory(key: string): MOECategory {
  return MOE_TAXONOMY[key]?.category ?? FALLBACK_CATEGORY;
}
