/** Formats a local Date as YYYY-MM-DD (no UTC conversion). */
export function toYYYYMMDD(date: Date): string {
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`;
}

/** Converts an HH:MM string to total minutes since midnight. */
export function toMinutes(hhmm: string): number {
  const [h, m] = hhmm.split(':').map(Number);
  return h * 60 + m;
}

/** Height in pixels of a one-hour time slot (matches Tailwind h-20 = 5rem = 80px). */
export const SLOT_HEIGHT_PX = 80;

/** Granularity of one time slot in minutes (15 min = 4 slots per hour). */
export const SLOT_STEP_MIN = 15;

/**
 * Snaps a start time (in minutes) DOWN to the nearest SLOT_STEP_MIN boundary.
 * e.g. 601 → 600 (10:01 → 10:00)
 */
export function snapStart(minutes: number): number {
  return Math.floor(minutes / SLOT_STEP_MIN) * SLOT_STEP_MIN;
}

/**
 * Snaps an end time (in minutes) UP to the nearest SLOT_STEP_MIN boundary,
 * guaranteeing at least one full slot (SLOT_STEP_MIN) after the snapped start.
 * e.g. end=604, snappedStart=600 → 615 (10:04 → 10:15)
 */
export function snapEnd(minutes: number, snappedStart: number): number {
  const ceiled = Math.ceil(minutes / SLOT_STEP_MIN) * SLOT_STEP_MIN;
  return Math.max(ceiled, snappedStart + SLOT_STEP_MIN);
}

export interface EventLayout {
  id: string;
  col: number;
  totalCols: number;
}

/**
 * Assigns a column index and total-column-count to each event so that
 * overlapping events are displayed side-by-side rather than stacked.
 *
 * Overlap detection uses snapped times (15-min slots) so that two events
 * whose snapped ranges share a slot are correctly treated as concurrent.
 */
export function layoutEvents(
  events: Array<{ id: string; startTime: string; endTime: string }>
): EventLayout[] {
  if (events.length === 0) return [];

  const sorted = [...events].sort(
    (a, b) => toMinutes(a.startTime) - toMinutes(b.startTime)
  );

  // Pre-compute snapped ranges used for all overlap checks
  const snapped = sorted.map((e) => {
    const ss = snapStart(toMinutes(e.startTime));
    const se = snapEnd(toMinutes(e.endTime), ss);
    return { ss, se };
  });

  const layouts: EventLayout[] = sorted.map((e) => ({
    id: e.id,
    col: 0,
    totalCols: 1,
  }));

  // Greedily place each event in the first column whose last event has ended
  const colEnds: number[] = [];
  for (let i = 0; i < layouts.length; i++) {
    const { ss, se } = snapped[i];
    let placed = -1;
    for (let c = 0; c < colEnds.length; c++) {
      if (colEnds[c] <= ss) {
        placed = c;
        break;
      }
    }
    if (placed === -1) {
      placed = colEnds.length;
      colEnds.push(se);
    } else {
      colEnds[placed] = se;
    }
    layouts[i].col = placed;
  }

  // totalCols = highest column index among concurrent events + 1
  for (let i = 0; i < layouts.length; i++) {
    const { ss: si, se: ei } = snapped[i];
    let maxCol = layouts[i].col;
    for (let j = 0; j < layouts.length; j++) {
      const { ss: sj, se: ej } = snapped[j];
      if (sj < ei && ej > si) {
        maxCol = Math.max(maxCol, layouts[j].col);
      }
    }
    layouts[i].totalCols = maxCol + 1;
  }

  return layouts;
}
