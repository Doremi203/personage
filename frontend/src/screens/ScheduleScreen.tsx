import { useState, useEffect, useCallback, useRef } from 'react';
import { ChevronLeft, ChevronRight, Clock } from 'lucide-react';
import {
  listTasks,
  ApiTaskItem,
  ApiTaskPriority,
  ApiTaskCategory,
} from '../utils/taskerService';
import { toYYYYMMDD, toMinutes, layoutEvents } from '../utils/dateUtils';

// ─── Design tokens ────────────────────────────────────────────
const tokens = {
  bg:       'oklch(0.985 0.008 80)',
  surface:  'oklch(1 0 0)',
  subtle:   'oklch(0.96 0.008 75)',
  hairline: 'oklch(0.91 0.01 75)',
  divider:  'oklch(0.94 0.01 75)',
  ink:      'oklch(0.22 0.015 60)',
  ink2:     'oklch(0.42 0.015 60)',
  ink3:     'oklch(0.58 0.012 65)',
  ink4:     'oklch(0.72 0.01 65)',
  // priority
  high:     { fill: 'oklch(0.94 0.04 40)',   rail: 'oklch(0.62 0.14 35)',  ink: 'oklch(0.38 0.12 35)'  },
  medium:   { fill: 'oklch(0.93 0.04 260)',  rail: 'oklch(0.58 0.14 265)', ink: 'oklch(0.38 0.14 265)' },
  low:      { fill: 'oklch(0.94 0.035 150)', rail: 'oklch(0.60 0.12 155)', ink: 'oklch(0.38 0.10 155)' },
  // category
  work:     { fill: 'oklch(0.93 0.04 260)',  rail: 'oklch(0.58 0.14 265)', ink: 'oklch(0.38 0.14 265)' },
  study:    { fill: 'oklch(0.94 0.04 70)',   rail: 'oklch(0.64 0.12 70)',  ink: 'oklch(0.40 0.10 70)'  },
  personal: { fill: 'oklch(0.94 0.035 150)', rail: 'oklch(0.60 0.12 155)', ink: 'oklch(0.38 0.10 155)' },
  now:    'oklch(0.60 0.18 30)',
} as const;

const HOUR_PX            = 62;
const DEFAULT_GRID_START = 7;
const DEFAULT_GRID_END   = 22;

const SERIF = '"Instrument Serif", "Iowan Old Style", Georgia, "Times New Roman", serif';
const SANS  = '"Inter", -apple-system, system-ui, "Segoe UI", sans-serif';

const RU_WEEKDAYS   = ['пн', 'вт', 'ср', 'чт', 'пт', 'сб', 'вс'];
const RU_MONTHS_NOM = ['Январь','Февраль','Март','Апрель','Май','Июнь','Июль','Август','Сентябрь','Октябрь','Ноябрь','Декабрь'];
const RU_MONTHS_GEN = ['января','февраля','марта','апреля','мая','июня','июля','августа','сентября','октября','ноября','декабря'];

// ─── Types ────────────────────────────────────────────────────
export interface ScheduleEvent {
  id: string;
  title: string;
  startTime: string;
  endTime: string;
  date: string;
  priority: 'high' | 'medium' | 'low';
  category: string;
}

// ─── Helpers ──────────────────────────────────────────────────
function toHHMM(isoTimestamp: string): string {
  const d = new Date(isoTimestamp);
  return `${d.getHours().toString().padStart(2, '0')}:${d.getMinutes().toString().padStart(2, '0')}`;
}

function toApiDateParam(date: Date): string {
  const d = date.getDate().toString().padStart(2, '0');
  const m = (date.getMonth() + 1).toString().padStart(2, '0');
  return `${d}-${m}-${date.getFullYear()}`;
}

function mapPriority(priority: string): 'high' | 'medium' | 'low' {
  if (priority === ApiTaskPriority.HIGH) return 'high';
  if (priority === ApiTaskPriority.LOW) return 'low';
  return 'medium';
}

function mapCategory(category: string): string {
  if (category === ApiTaskCategory.WORK) return 'work';
  if (category === ApiTaskCategory.STUDY) return 'study';
  return 'personal';
}

function addMinutesToHHMM(hhmm: string, mins: number): string {
  const total = toMinutes(hhmm) + mins;
  return `${Math.floor(total / 60).toString().padStart(2, '0')}:${(total % 60).toString().padStart(2, '0')}`;
}

function getGridBounds(events: Pick<ScheduleEvent, 'startTime' | 'endTime'>[], nowMins?: number) {
  let startHour = DEFAULT_GRID_START;
  let endHour = DEFAULT_GRID_END;

  if (events.length > 0) {
    const mins = events.flatMap((event) => [toMinutes(event.startTime), toMinutes(event.endTime)]);
    startHour = Math.min(startHour, Math.floor(Math.min(...mins) / 60));
    endHour = Math.max(endHour, Math.ceil(Math.max(...mins) / 60));
  }

  if (nowMins !== undefined) {
    startHour = Math.min(startHour, Math.floor(nowMins / 60));
    endHour = Math.max(endHour, Math.ceil((nowMins + 1) / 60));
  }

  startHour = Math.max(0, startHour);
  endHour = Math.min(24, Math.max(startHour + 1, endHour));

  return { startHour, endHour };
}

function mapApiTaskToEvent(task: ApiTaskItem): ScheduleEvent | null {
  if (!task.startTime) return null;
  const startHHMM = toHHMM(task.startTime);
  const endHHMM = task.endTime ? toHHMM(task.endTime) : addMinutesToHHMM(startHHMM, 30);
  const date = toYYYYMMDD(new Date(task.startTime));
  return {
    id: task.id,
    title: task.title,
    startTime: startHHMM,
    endTime: endHHMM,
    date,
    priority: mapPriority(task.priority),
    category: mapCategory(task.category),
  };
}

function getMondayOf(date: Date): Date {
  const d = new Date(date);
  d.setDate(d.getDate() - ((d.getDay() + 6) % 7));
  d.setHours(0, 0, 0, 0);
  return d;
}

function formatDuration(start: string, end: string): string {
  const mins = toMinutes(end) - toMinutes(start);
  if (mins <= 0) return '';
  const h = Math.floor(mins / 60);
  const m = mins % 60;
  if (h > 0 && m > 0) return `${h} ч ${m} мин`;
  if (h > 0) return `${h} ч`;
  return `${m} мин`;
}

// ─── WeekStrip ────────────────────────────────────────────────
interface WeekStripProps {
  weekStart: Date;
  selectedDate: Date;
  events: ScheduleEvent[];
  onSelect: (d: Date) => void;
}

function WeekStrip({ weekStart, selectedDate, events, onSelect }: WeekStripProps) {
  const today = new Date();
  return (
    <div style={{
      display: 'flex', gap: 4,
      padding: '4px 12px 12px',
      background: tokens.bg,
      flexShrink: 0,
    }}>
      {RU_WEEKDAYS.map((label, i) => {
        const day = new Date(weekStart);
        day.setDate(weekStart.getDate() + i);
        const isSelected = toYYYYMMDD(day) === toYYYYMMDD(selectedDate);
        const isToday = toYYYYMMDD(day) === toYYYYMMDD(today);
        const dayEvents = events.filter(e => e.date === toYYYYMMDD(day));

        return (
          <button
            key={i}
            onClick={() => onSelect(new Date(day))}
            style={{
              flex: 1,
              display: 'flex', flexDirection: 'column', alignItems: 'center',
              gap: 4,
              background: isSelected ? tokens.ink : 'transparent',
              border: 'none',
              borderRadius: 14,
              padding: '8px 2px',
              cursor: 'pointer',
              fontFamily: SANS,
              minWidth: 0,
            }}
          >
            <span style={{
              fontSize: 10,
              fontWeight: 500,
              textTransform: 'uppercase',
              letterSpacing: 0.5,
              color: isSelected ? 'rgba(255,255,255,0.7)' : tokens.ink4,
            }}>
              {label}
            </span>

            <span style={{
              fontFamily: SERIF,
              fontSize: 22,
              fontWeight: 400,
              lineHeight: 1,
              color: isSelected ? tokens.bg : (isToday ? tokens.now : tokens.ink),
              position: 'relative',
              display: 'inline-block',
            }}>
              {day.getDate()}
              {isToday && !isSelected && (
                <span style={{
                  position: 'absolute', bottom: -4, left: '50%',
                  transform: 'translateX(-50%)',
                  width: 4, height: 4, borderRadius: '50%',
                  background: tokens.now,
                  display: 'block',
                }} />
              )}
            </span>

            <div style={{ display: 'flex', gap: 2, height: 4, marginTop: 2 }}>
              {dayEvents.slice(0, 3).map((e, idx) => (
                <div key={idx} style={{
                  width: 4, height: 4, borderRadius: '50%',
                  background: isSelected
                    ? 'rgba(255,255,255,0.85)'
                    : tokens[e.priority as 'high' | 'medium' | 'low'].rail,
                }} />
              ))}
            </div>
          </button>
        );
      })}
    </div>
  );
}

// ─── HourGrid ─────────────────────────────────────────────────
interface HourGridProps {
  date: Date;
  events: ScheduleEvent[];
  isToday: boolean;
  onEventClick: (e: ScheduleEvent) => void;
  scrollRef: React.RefObject<HTMLDivElement | null>;
}

function HourGrid({ date, events, isToday, onEventClick, scrollRef }: HourGridProps) {
  const [nowMins, setNowMins] = useState(() => {
    const n = new Date();
    return n.getHours() * 60 + n.getMinutes();
  });

  const dayEvents = events.filter(e => e.date === toYYYYMMDD(date));
  const { startHour: gridStart, endHour: gridEnd } = getGridBounds(dayEvents, isToday ? nowMins : undefined);
  const hours = Array.from({ length: gridEnd - gridStart }, (_, i) => i + gridStart);
  const gridHeight = hours.length * HOUR_PX;

  useEffect(() => {
    const id = setInterval(() => {
      const n = new Date();
      setNowMins(n.getHours() * 60 + n.getMinutes());
    }, 60_000);
    return () => clearInterval(id);
  }, []);

  // Scroll so 08:00 is near the top on first render of this grid
  useEffect(() => {
    if (scrollRef.current) {
      scrollRef.current.scrollTop = Math.max(0, (Math.max(8, gridStart) - gridStart) * HOUR_PX - 10);
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [date, gridStart]);

  const layouts = layoutEvents(dayEvents);

  const showNow = isToday && nowMins >= gridStart * 60 && nowMins <= gridEnd * 60;
  const nowTop = (nowMins - gridStart * 60) * (HOUR_PX / 60);
  const nowLabel = `${Math.floor(nowMins / 60).toString().padStart(2, '0')}:${(nowMins % 60).toString().padStart(2, '0')}`;

  return (
    <div
      ref={scrollRef}
      style={{
        flex: 1,
        overflowY: 'auto',
        overflowX: 'hidden',
        background: tokens.bg,
        WebkitOverflowScrolling: 'touch',
      } as React.CSSProperties}
    >
      <div style={{ display: 'flex', padding: '8px 0 40px' }}>
        {/* Hour labels */}
        <div style={{ width: 52, flexShrink: 0, paddingTop: 6 }}>
          {hours.map(h => (
            <div key={h} style={{
              height: HOUR_PX,
              paddingRight: 8,
              textAlign: 'right',
              fontFamily: SANS,
              fontSize: 11,
              fontWeight: 500,
              color: tokens.ink4,
              fontVariantNumeric: 'tabular-nums',
              transform: 'translateY(-6px)',
            }}>
              {h.toString().padStart(2, '0')}:00
            </div>
          ))}
        </div>

        {/* Event column */}
        <div style={{ flex: 1, position: 'relative', marginRight: 12 }}>
          {/* Hour lines */}
           {hours.map((_, i) => (
             <div key={i} style={{
               position: 'absolute',
               top: i * HOUR_PX,
              left: 0, right: 0,
              height: 1,
              background: tokens.divider,
            }} />
          ))}
          {/* Half-hour ticks */}
           {hours.slice(0, -1).map((_, i) => (
             <div key={`h-${i}`} style={{
               position: 'absolute',
               top: i * HOUR_PX + HOUR_PX / 2,
              left: 0,
              width: 6,
              height: 1,
              background: tokens.divider,
            }} />
          ))}

          {/* Spacer */}
           <div style={{ height: gridHeight }} />

           {/* Events */}
           {layouts.map(({ id, col, totalCols }) => {
             const ev = dayEvents.find(e => e.id === id);
             if (!ev) return null;
             const startMins = toMinutes(ev.startTime);
             const endMins   = toMinutes(ev.endTime);
             const rawTop    = (startMins - gridStart * 60) * (HOUR_PX / 60);
             const rawBottom = (endMins   - gridStart * 60) * (HOUR_PX / 60);
             const clampedTop    = Math.max(0, rawTop);
             const clampedBottom = Math.min(gridHeight, rawBottom);
             const height = Math.max(0, clampedBottom - clampedTop - 2);
             if (height === 0) return null;
             const visualHeight = Math.max(height, 20);
             const top = Math.min(clampedTop, gridHeight - visualHeight);
             const isCompact = visualHeight < 44;

             const pct = 100 / totalCols;
             const pal = tokens[ev.priority];

            return (
              <div
                key={ev.id}
                onClick={() => onEventClick(ev)}
                 title={`${ev.title} · ${ev.startTime}–${ev.endTime}`}
                 style={{
                   position: 'absolute',
                   top,
                   height: visualHeight,
                   left: `calc(${col * pct}% + 2px)`,
                   width: `calc(${pct}% - 4px)`,
                   background: pal.fill,
                  borderLeft: `3px solid ${pal.rail}`,
                   color: pal.ink,
                   borderRadius: 6,
                   padding: isCompact ? '2px 5px' : '3px 6px',
                   overflow: 'hidden',
                   cursor: 'pointer',
                   boxSizing: 'border-box',
                   fontFamily: SANS,
                   display: 'flex',
                   alignItems: isCompact ? 'center' : undefined,
                 }}
               >
                 {isCompact ? (
                   <div style={{
                     minWidth: 0,
                     fontSize: 10.5,
                     fontWeight: 600,
                     lineHeight: 1.1,
                     whiteSpace: 'nowrap',
                     overflow: 'hidden',
                     textOverflow: 'ellipsis',
                   }}>
                     <span style={{ fontVariantNumeric: 'tabular-nums', opacity: 0.85, marginRight: 4 }}>
                       {ev.startTime}
                     </span>
                     {ev.title}
                   </div>
                 ) : (
                   <>
                     <div style={{
                       fontSize: 11.5,
                       fontWeight: 500,
                       fontVariantNumeric: 'tabular-nums',
                       opacity: 0.85,
                       marginBottom: 1,
                     }}>
                       {ev.startTime}
                     </div>
                     <div style={{
                       fontSize: 12.5,
                       fontWeight: 600,
                       lineHeight: 1.25,
                       display: '-webkit-box',
                       WebkitLineClamp: 2,
                       WebkitBoxOrient: 'vertical',
                       overflow: 'hidden',
                       textOverflow: 'ellipsis',
                     } as React.CSSProperties}>
                       {ev.title}
                     </div>
                   </>
                 )}
               </div>
             );
           })}

          {/* Now indicator */}
          {showNow && (
            <div style={{
               position: 'absolute',
               top: nowTop,
              left: -56, right: 0,
              zIndex: 10,
              pointerEvents: 'none',
            }}>
              <div style={{ display: 'flex', alignItems: 'center', transform: 'translateY(-50%)' }}>
                <div style={{
                  fontFamily: SANS, fontSize: 10.5, fontWeight: 600,
                  color: tokens.now,
                  background: tokens.bg,
                  padding: '1px 4px',
                  borderRadius: 4,
                  fontVariantNumeric: 'tabular-nums',
                  marginRight: 2,
                  flexShrink: 0,
                }}>
                  {nowLabel}
                </div>
                <div style={{
                  width: 7, height: 7, borderRadius: '50%',
                  background: tokens.now,
                  marginLeft: -2,
                  flexShrink: 0,
                }} />
                <div style={{ flex: 1, height: 1.5, background: tokens.now }} />
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

// ─── EventSheet ───────────────────────────────────────────────
interface EventSheetProps {
  event: ScheduleEvent | null;
  onClose: () => void;
}

function EventSheet({ event, onClose }: EventSheetProps) {
  const [mounted, setMounted] = useState(false);
  const [visible, setVisible] = useState(false);

  useEffect(() => {
    if (event) {
      const raf1 = requestAnimationFrame(() => {
        setMounted(true);
        requestAnimationFrame(() => setVisible(true));
      });
      return () => cancelAnimationFrame(raf1);
    } else {
      const raf2 = requestAnimationFrame(() => setVisible(false));
      const t = setTimeout(() => setMounted(false), 320);
      return () => { cancelAnimationFrame(raf2); clearTimeout(t); };
    }
  }, [event]);

  if (!mounted || !event) return null;

  const pal = tokens[event.priority];
  const [y, m, d] = event.date.split('-').map(Number);
  const dateObj = new Date(y, m - 1, d);
  const weekday = RU_WEEKDAYS[(dateObj.getDay() + 6) % 7];
  const dateLine = `${d} ${RU_MONTHS_GEN[m - 1]}, ${weekday}`;
  const duration = formatDuration(event.startTime, event.endTime);

  const prioLabel = { high: 'Высокий', medium: 'Средний', low: 'Низкий' }[event.priority];
  const catLabel  = { work: 'Работа', study: 'Учёба', personal: 'Личное' }[event.category as 'work' | 'study' | 'personal'] ?? event.category;
  const catPal = tokens[event.category as 'work' | 'study' | 'personal'] ?? tokens.work;

  return (
    <>
      <div
        onClick={onClose}
        className={visible ? 'animate-fade-in' : ''}
        style={{
          position: 'fixed', inset: 0,
          background: 'rgba(25, 20, 15, 0.38)',
          zIndex: 50,
          opacity: visible ? undefined : 0,
        }}
      />
      <div
        className={visible ? 'animate-sheet-up' : ''}
        style={{
          position: 'fixed', bottom: 0, left: 0, right: 0,
          background: tokens.surface,
          borderTopLeftRadius: 28, borderTopRightRadius: 28,
          padding: '12px 20px 40px',
          boxShadow: '0 -12px 40px rgba(40,30,20,0.18)',
          zIndex: 51,
          fontFamily: SANS,
          transform: visible ? undefined : 'translateY(100%)',
        }}
      >
        {/* Drag handle */}
        <div style={{
          width: 36, height: 4, borderRadius: 2,
          background: tokens.hairline,
          margin: '0 auto 16px',
        }} />

        {/* Color strip */}
        <div style={{
          width: 34, height: 4, borderRadius: 2,
          background: pal.rail, marginBottom: 14,
        }} />

        {/* Title */}
        <div style={{
          fontFamily: SERIF,
          fontSize: 26, fontWeight: 400,
          color: tokens.ink,
          lineHeight: 1.2, marginBottom: 4,
          letterSpacing: -0.2,
        }}>
          {event.title}
        </div>

        {/* Date */}
        <div style={{ fontSize: 13, color: tokens.ink3, marginBottom: 18 }}>
          {dateLine}
        </div>

        {/* Time row */}
        <div style={{
          display: 'flex', alignItems: 'center', gap: 12,
          padding: '12px 14px',
          background: tokens.subtle, borderRadius: 12, marginBottom: 10,
        }}>
          <Clock size={16} color={tokens.ink3} strokeWidth={1.5} />
          <div style={{ flex: 1 }}>
            <div style={{
              fontSize: 14, fontWeight: 600, color: tokens.ink,
              fontVariantNumeric: 'tabular-nums',
            }}>
              {event.startTime} – {event.endTime}
            </div>
            {duration && (
              <div style={{ fontSize: 12, color: tokens.ink3 }}>{duration}</div>
            )}
          </div>
        </div>

        {/* Meta chips */}
        <div style={{ display: 'flex', gap: 10 }}>
          <div style={{
            flex: 1, background: tokens.subtle, borderRadius: 12, padding: '10px 14px',
          }}>
            <div style={{
              fontSize: 10.5, fontWeight: 500, letterSpacing: 0.6,
              textTransform: 'uppercase', color: tokens.ink4, marginBottom: 4,
            }}>
              Категория
            </div>
            <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
              <div style={{ width: 7, height: 7, borderRadius: '50%', background: catPal.rail }} />
              <div style={{ fontSize: 13.5, fontWeight: 600, color: tokens.ink }}>{catLabel}</div>
            </div>
          </div>
          <div style={{
            flex: 1, background: tokens.subtle, borderRadius: 12, padding: '10px 14px',
          }}>
            <div style={{
              fontSize: 10.5, fontWeight: 500, letterSpacing: 0.6,
              textTransform: 'uppercase', color: tokens.ink4, marginBottom: 4,
            }}>
              Приоритет
            </div>
            <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
              <div style={{ width: 7, height: 7, borderRadius: '50%', background: pal.rail }} />
              <div style={{ fontSize: 13.5, fontWeight: 600, color: tokens.ink }}>{prioLabel}</div>
            </div>
          </div>
        </div>
      </div>
    </>
  );
}

// ─── ScheduleScreen ───────────────────────────────────────────
const ScheduleScreen = () => {
  const today = new Date();
  const [selectedDate, setSelectedDate] = useState<Date>(today);
  const [weekStart, setWeekStart]       = useState<Date>(() => getMondayOf(today));
  const [events, setEvents]             = useState<ScheduleEvent[]>([]);
  const [loading, setLoading]           = useState(false);
  const [error, setError]               = useState<string | null>(null);
  const [sheet, setSheet]               = useState<ScheduleEvent | null>(null);
  const scrollRef = useRef<HTMLDivElement | null>(null);

  const fetchEvents = useCallback(async () => {
      setLoading(true);
      setError(null);
      try {
        const till = new Date(weekStart);
        till.setDate(weekStart.getDate() + 6);
       const pageSize = 100;
       const allTasks: ApiTaskItem[] = [];
       let page = 1;

       while (true) {
         const response = await listTasks({
           from: toApiDateParam(weekStart),
           till: toApiDateParam(till),
           pageSize,
           page,
         });
         const tasks = response.tasks ?? [];
         allTasks.push(...tasks);

         if (tasks.length < pageSize || allTasks.length >= response.total) {
           break;
         }

         page += 1;
       }

       const mapped = allTasks
         .map(mapApiTaskToEvent)
         .filter((e): e is ScheduleEvent => e !== null);
       setEvents(mapped);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Не удалось загрузить расписание');
    } finally {
      setLoading(false);
    }
  }, [weekStart]);

  useEffect(() => {
    void fetchEvents();
  }, [fetchEvents]);

  const goToPrevWeek = () => {
    const newWeek = new Date(weekStart);
    newWeek.setDate(weekStart.getDate() - 7);
    setWeekStart(newWeek);
    const newSelected = new Date(selectedDate);
    newSelected.setDate(selectedDate.getDate() - 7);
    setSelectedDate(newSelected);
  };

  const goToNextWeek = () => {
    const newWeek = new Date(weekStart);
    newWeek.setDate(weekStart.getDate() + 7);
    setWeekStart(newWeek);
    const newSelected = new Date(selectedDate);
    newSelected.setDate(selectedDate.getDate() + 7);
    setSelectedDate(newSelected);
  };

  const goToToday = () => {
    const now = new Date();
    setSelectedDate(now);
    setWeekStart(getMondayOf(now));
  };

  const monthName = RU_MONTHS_NOM[selectedDate.getMonth()];
  const year = selectedDate.getFullYear();
  const isToday = toYYYYMMDD(selectedDate) === toYYYYMMDD(new Date());

  const iconBtn: React.CSSProperties = {
    width: 34, height: 34, borderRadius: 10,
    background: tokens.surface,
    border: `1px solid ${tokens.hairline}`,
    display: 'flex', alignItems: 'center', justifyContent: 'center',
    cursor: 'pointer', color: tokens.ink2,
    flexShrink: 0,
  };

  return (
    <div
      className="pt-16 md:pt-0"
      style={{
        background: tokens.bg,
        display: 'flex', flexDirection: 'column',
        height: '100%', overflow: 'hidden',
        color: tokens.ink, fontFamily: SANS,
      }}
    >
      {/* Top bar */}
      <div style={{
        display: 'flex', alignItems: 'center', justifyContent: 'space-between',
        padding: '16px 16px 6px',
        flexShrink: 0,
      }}>
        <div style={{ display: 'flex', alignItems: 'baseline', gap: 6 }}>
          <span style={{ fontFamily: SERIF, fontSize: 26, color: tokens.ink, lineHeight: 1 }}>
            {monthName}
          </span>
          <span style={{ fontFamily: SERIF, fontStyle: 'italic', fontSize: 22, color: tokens.ink3, lineHeight: 1 }}>
            {year}
          </span>
        </div>

        <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
          <button onClick={goToPrevWeek} style={iconBtn} aria-label="Предыдущая неделя">
            <ChevronLeft size={16} />
          </button>
          <button
            onClick={goToToday}
            style={{
              fontFamily: SANS, fontSize: 13, fontWeight: 500,
              color: tokens.ink,
              background: tokens.surface,
              border: `1px solid ${tokens.hairline}`,
              borderRadius: 999, padding: '6px 14px', cursor: 'pointer',
            }}
          >
            Сегодня
          </button>
          <button onClick={goToNextWeek} style={iconBtn} aria-label="Следующая неделя">
            <ChevronRight size={16} />
          </button>
        </div>
      </div>

      {/* Priority legend */}
      <div style={{
        display: 'flex', alignItems: 'center', gap: 14,
        padding: '2px 16px 8px',
        flexShrink: 0,
      }}>
        {(['high', 'medium', 'low'] as const).map(p => (
          <div key={p} style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
            <div style={{ width: 7, height: 7, borderRadius: '50%', background: tokens[p].rail }} />
            <span style={{ fontSize: 11, color: tokens.ink3, fontWeight: 500 }}>
              {p === 'high' ? 'Высокий' : p === 'medium' ? 'Средний' : 'Низкий'}
            </span>
          </div>
        ))}
      </div>

      {/* Week strip */}
      <WeekStrip
        weekStart={weekStart}
        selectedDate={selectedDate}
        events={events}
        onSelect={setSelectedDate}
      />

      {/* Hour grid or loading/error state */}
      {loading ? (
        <div style={{
          flex: 1, display: 'flex', alignItems: 'center', justifyContent: 'center',
          color: tokens.ink3, fontSize: 14,
        }}>
          Загрузка…
        </div>
      ) : error ? (
        <div style={{
          flex: 1, display: 'flex', flexDirection: 'column',
          alignItems: 'center', justifyContent: 'center', gap: 12,
        }}>
          <span style={{ color: 'oklch(0.55 0.14 35)', fontSize: 14 }}>{error}</span>
          <button
            onClick={() => void fetchEvents()}
            style={{
              fontFamily: SANS, fontSize: 13, fontWeight: 500,
              color: tokens.surface, background: tokens.ink,
              border: 'none', borderRadius: 8, padding: '8px 18px', cursor: 'pointer',
            }}
          >
            Повторить
          </button>
        </div>
      ) : (
        <HourGrid
          date={selectedDate}
          events={events}
          isToday={isToday}
          onEventClick={setSheet}
          scrollRef={scrollRef}
        />
      )}

      {/* Event detail sheet */}
      <EventSheet event={sheet} onClose={() => setSheet(null)} />
    </div>
  );
};

export default ScheduleScreen;
