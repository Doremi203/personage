// Mobile design tokens — warm + amber palette (ported from design bundle's
// mobile-tokens.js / Personage Mobile.html). Single source of truth for the
// mobile UI; do not duplicate these colors inline.

export interface PriorityScale {
  fill: string;
  rail: string;
  ink: string;
}

export const T = {
  bg:       'oklch(0.985 0.008 80)',
  bgDeep:   'oklch(0.965 0.01 78)',
  surface:  'oklch(1 0 0)',
  subtle:   'oklch(0.96 0.008 75)',
  subtleHi: 'oklch(0.94 0.01 75)',
  hairline: 'oklch(0.91 0.01 75)',
  divider:  'oklch(0.94 0.01 75)',

  ink:      'oklch(0.22 0.015 60)',
  ink2:     'oklch(0.42 0.015 60)',
  ink3:     'oklch(0.58 0.012 65)',
  ink4:     'oklch(0.72 0.01 65)',

  sideBg:   'oklch(0.20 0.014 55)',
  sideBg2:  'oklch(0.16 0.012 55)',
  sideInk:  'oklch(0.96 0.008 75)',
  sideInk2: 'oklch(0.72 0.012 70)',

  amber:    'oklch(0.74 0.15 75)',
  amberDp:  'oklch(0.66 0.16 70)',
  amberHi:  'oklch(0.82 0.12 80)',
  amberFill:'oklch(0.94 0.04 75)',
  amberInk: 'oklch(0.40 0.10 65)',

  high:     { fill: 'oklch(0.94 0.04 40)',   rail: 'oklch(0.62 0.14 35)',  ink: 'oklch(0.38 0.12 35)'  } as PriorityScale,
  medium:   { fill: 'oklch(0.94 0.04 70)',   rail: 'oklch(0.66 0.14 70)',  ink: 'oklch(0.40 0.12 70)' } as PriorityScale,
  low:      { fill: 'oklch(0.94 0.035 150)', rail: 'oklch(0.60 0.10 155)', ink: 'oklch(0.38 0.08 155)' } as PriorityScale,
  work:     { fill: 'oklch(0.93 0.04 260)',  rail: 'oklch(0.55 0.10 265)', ink: 'oklch(0.38 0.10 265)' } as PriorityScale,
  study:    { fill: 'oklch(0.94 0.04 70)',   rail: 'oklch(0.66 0.13 70)',  ink: 'oklch(0.40 0.10 70)'  } as PriorityScale,
  personal: { fill: 'oklch(0.94 0.035 150)', rail: 'oklch(0.60 0.10 155)', ink: 'oklch(0.38 0.08 155)' } as PriorityScale,

  now:        'oklch(0.60 0.18 30)',
  danger:     'oklch(0.55 0.16 30)',
  dangerFill: 'oklch(0.95 0.03 30)',
  ok:         'oklch(0.55 0.10 155)',
  okFill:     'oklch(0.94 0.035 150)',
  info:       'oklch(0.55 0.10 265)',
  infoFill:   'oklch(0.93 0.04 260)',
} as const;

export const SERIF = '"Instrument Serif", "Iowan Old Style", Georgia, "Times New Roman", serif';
export const SANS  = '"Inter", -apple-system, system-ui, "Segoe UI", sans-serif';

export type Priority = 'high' | 'medium' | 'low';
export type Category = 'work' | 'study' | 'personal';

export function priorityScale(p: Priority): PriorityScale {
  return T[p];
}

export function categoryScale(c: Category): PriorityScale {
  return T[c];
}

export const PRIORITY_LABELS: Record<Priority, string> = {
  high:   'Высокий',
  medium: 'Средний',
  low:    'Низкий',
};

export const CATEGORY_LABELS: Record<Category, string> = {
  work:     'Работа',
  study:    'Учёба',
  personal: 'Личное',
};
