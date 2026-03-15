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

export interface EventLayout {
  id: string;
  col: number;
  totalCols: number;
}

/**
 * Assigns a column index and total-column-count to each event so that
 * overlapping events are displayed side-by-side rather than stacked.
 *
 * Events are identified by their id, startTime, and endTime strings (HH:MM).
 */
export function layoutEvents(
  events: Array<{ id: string; startTime: string; endTime: string }>
): EventLayout[] {
  if (events.length === 0) return [];

  const sorted = [...events].sort(
    (a, b) => toMinutes(a.startTime) - toMinutes(b.startTime)
  );

  const layouts: EventLayout[] = sorted.map((e) => ({
    id: e.id,
    col: 0,
    totalCols: 1,
  }));

  // Greedily place each event in the first column whose last event has ended
  const colEnds: number[] = [];
  for (let i = 0; i < layouts.length; i++) {
    const start = toMinutes(sorted[i].startTime);
    const end = Math.max(toMinutes(sorted[i].endTime), start + 1);
    let placed = -1;
    for (let c = 0; c < colEnds.length; c++) {
      if (colEnds[c] <= start) {
        placed = c;
        break;
      }
    }
    if (placed === -1) {
      placed = colEnds.length;
      colEnds.push(end);
    } else {
      colEnds[placed] = end;
    }
    layouts[i].col = placed;
  }

  // totalCols = highest column index among concurrent events + 1
  for (let i = 0; i < layouts.length; i++) {
    const si = toMinutes(sorted[i].startTime);
    const ei = Math.max(toMinutes(sorted[i].endTime), si + 1);
    let maxCol = layouts[i].col;
    for (let j = 0; j < layouts.length; j++) {
      const sj = toMinutes(sorted[j].startTime);
      const ej = Math.max(toMinutes(sorted[j].endTime), sj + 1);
      if (sj < ei && ej > si) {
        maxCol = Math.max(maxCol, layouts[j].col);
      }
    }
    layouts[i].totalCols = maxCol + 1;
  }

  return layouts;
}
