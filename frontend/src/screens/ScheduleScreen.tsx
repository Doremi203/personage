import { useCallback, useEffect, useMemo, useState } from 'react';
import { ChevronLeft, ChevronRight, CalendarOff } from 'lucide-react';
import {
  CATEGORY_LABELS,
  SANS,
  SERIF,
  T,
  type Category,
  type Priority,
} from '../mobile/tokens';
import { TaskDetailSheet, type DetailTask } from '../mobile/TaskDetailSheet';
import { ErrorState } from '../mobile/StateViews';
import {
  ApiTaskCategory,
  ApiTaskPriority,
  ApiTaskStatus,
  completeTask,
  deleteTask,
  listTasks,
  postponeTask,
  updateTask,
  type ApiTaskItem,
  type UpdateTaskPatch,
} from '../utils/taskerService';
import {
  RU_MONTHS_GEN,
  RU_WEEKDAYS_SHORT,
  sameDay,
  startOfDay,
  toApiDateParam,
  toHHMM,
} from '../utils/dateFormat';

interface ScheduleEvent {
  id: string;
  title: string;
  startTime: Date;
  endTime: Date;
  priority: Priority;
  category: Category;
  raw: ApiTaskItem;
}

function getMondayOf(d: Date): Date {
  const x = startOfDay(d);
  const dow = (x.getDay() + 6) % 7; // Mon=0
  x.setDate(x.getDate() - dow);
  return x;
}

function mapPriority(p: string): Priority {
  if (p === ApiTaskPriority.HIGH) return 'high';
  if (p === ApiTaskPriority.LOW)  return 'low';
  return 'medium';
}

function mapCategory(c: string): Category {
  if (c === ApiTaskCategory.WORK)  return 'work';
  if (c === ApiTaskCategory.STUDY) return 'study';
  return 'personal';
}

function mapStatus(s: string): DetailTask['status'] {
  if (s === ApiTaskStatus.COMPLETED) return 'completed';
  if (s === ApiTaskStatus.UNPLANNED) return 'unplanned';
  return 'planned';
}

function mapApiTaskToEvent(t: ApiTaskItem): ScheduleEvent | null {
  if (!t.startTime) return null;
  const start = new Date(t.startTime);
  if (Number.isNaN(start.getTime())) return null;
  const end = t.endTime ? new Date(t.endTime) : new Date(start.getTime() + 30 * 60 * 1000);
  return {
    id: t.id,
    title: t.title,
    startTime: start,
    endTime: end,
    priority: mapPriority(t.priority),
    category: mapCategory(t.category),
    raw: t,
  };
}

function formatBusy(events: ScheduleEvent[]): string {
  const totalMin = events.reduce((acc, e) => {
    const ms = e.endTime.getTime() - e.startTime.getTime();
    return acc + Math.max(0, Math.round(ms / 60000));
  }, 0);
  if (totalMin <= 0) return '';
  const h = Math.floor(totalMin / 60);
  const m = totalMin % 60;
  if (h > 0 && m > 0) return ` · ${h} ч ${m} мин занято`;
  if (h > 0) return ` · ${h} ч занято`;
  return ` · ${m} мин занято`;
}

function pluralEvents(n: number): string {
  const mod10 = n % 10;
  const mod100 = n % 100;
  if (mod100 >= 11 && mod100 <= 19) return `${n} событий`;
  if (mod10 === 1) return `${n} событие`;
  if (mod10 >= 2 && mod10 <= 4) return `${n} события`;
  return `${n} событий`;
}

const ScheduleScreen = () => {
  const today = useMemo(() => new Date(), []);
  const [selected, setSelected] = useState<Date>(today);
  const [weekStart, setWeekStart] = useState<Date>(() => getMondayOf(today));
  const [events, setEvents] = useState<ScheduleEvent[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [selectedEvent, setSelectedEvent] = useState<ScheduleEvent | null>(null);

  const fetchEvents = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const till = new Date(weekStart);
      till.setDate(weekStart.getDate() + 6);
      const pageSize = 100;
      const all: ApiTaskItem[] = [];
      let page = 1;
      while (true) {
        const resp = await listTasks({
          from: toApiDateParam(weekStart),
          till: toApiDateParam(till),
          pageSize,
          page,
        });
        const tasks = resp.tasks ?? [];
        all.push(...tasks);
        if (tasks.length < pageSize || all.length >= (resp.total ?? all.length)) break;
        page += 1;
      }
      setEvents(all.map(mapApiTaskToEvent).filter((e): e is ScheduleEvent => e !== null));
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Не удалось загрузить расписание');
    } finally {
      setLoading(false);
    }
  }, [weekStart]);

  useEffect(() => { void fetchEvents(); }, [fetchEvents]);

  const weekDays = useMemo(() => Array.from({ length: 7 }, (_, i) => {
    const d = new Date(weekStart);
    d.setDate(weekStart.getDate() + i);
    return d;
  }), [weekStart]);

  const eventsByDay = useMemo(() => {
    const map: Record<string, ScheduleEvent[]> = {};
    for (const e of events) {
      const key = startOfDay(e.startTime).toISOString();
      (map[key] ||= []).push(e);
    }
    Object.values(map).forEach((list) =>
      list.sort((a, b) => a.startTime.getTime() - b.startTime.getTime()),
    );
    return map;
  }, [events]);

  const dayEvents = eventsByDay[startOfDay(selected).toISOString()] ?? [];

  const isSelectedToday = sameDay(selected, today);
  const dateLabel = isSelectedToday
    ? `Сегодня · ${selected.getDate()} ${RU_MONTHS_GEN[selected.getMonth()]}`
    : `${RU_WEEKDAYS_SHORT[(selected.getDay() + 6) % 7]} · ${selected.getDate()} ${RU_MONTHS_GEN[selected.getMonth()]}`;

  const goPrevWeek = () => {
    const newSelected = new Date(selected);
    newSelected.setDate(selected.getDate() - 7);
    setSelected(newSelected);
    setWeekStart(getMondayOf(newSelected));
  };
  const goNextWeek = () => {
    const newSelected = new Date(selected);
    newSelected.setDate(selected.getDate() + 7);
    setSelected(newSelected);
    setWeekStart(getMondayOf(newSelected));
  };

  const handleAction = useCallback(
    async (id: string, action: (id: string) => Promise<unknown>) => {
      await action(id);
      setSelectedEvent(null);
      await fetchEvents();
    },
    [fetchEvents],
  );

  const handleSave = useCallback(
    async (id: string, patch: UpdateTaskPatch) => {
      await updateTask(id, patch);
      await fetchEvents();
    },
    [fetchEvents],
  );

  return (
    <>
      {/* Header — title + week nav on same line */}
      <div style={{ padding: '8px 20px 6px', background: T.bg }}>
        <div style={{
          display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 12,
        }}>
          <div style={{
            fontFamily: SERIF, fontSize: 34, lineHeight: 1.05,
            color: T.ink, letterSpacing: -0.5,
          }}>Расписание</div>
          <div style={{
            display: 'flex', alignItems: 'center', gap: 4,
            background: T.surface,
            border: `0.5px solid ${T.hairline}`,
            borderRadius: 999,
            padding: 4,
          }}>
            <button
              type="button"
              onClick={goPrevWeek}
              aria-label="Предыдущая неделя"
              style={{
                width: 28, height: 28, borderRadius: '50%',
                background: 'transparent', border: 'none',
                display: 'flex', alignItems: 'center', justifyContent: 'center',
                color: T.ink2, cursor: 'pointer', padding: 0,
              }}
            >
              <ChevronLeft size={17} strokeWidth={2} />
            </button>
            <div style={{
              fontSize: 12.5, fontWeight: 600, color: T.ink2,
              letterSpacing: 0.2, padding: '0 4px',
            }}>Неделя</div>
            <button
              type="button"
              onClick={goNextWeek}
              aria-label="Следующая неделя"
              style={{
                width: 28, height: 28, borderRadius: '50%',
                background: 'transparent', border: 'none',
                display: 'flex', alignItems: 'center', justifyContent: 'center',
                color: T.ink2, cursor: 'pointer', padding: 0,
              }}
            >
              <ChevronRight size={17} strokeWidth={2} />
            </button>
          </div>
        </div>
        <div style={{ fontSize: 13.5, color: T.ink3, marginTop: 4, lineHeight: 1.4 }}>
          {pluralEvents(events.length)}{formatBusy(events)}
        </div>
      </div>

      {/* Week strip */}
      <div style={{ padding: '4px 8px 14px' }}>
        <div style={{ display: 'flex', gap: 4 }}>
          {weekDays.map((d, i) => {
            const sel = sameDay(d, selected);
            const isToday2 = sameDay(d, today);
            const count = (eventsByDay[startOfDay(d).toISOString()] ?? []).length;
            return (
              <button
                key={i}
                type="button"
                onClick={() => setSelected(d)}
                style={{
                  flex: 1, background: 'transparent', border: 'none', cursor: 'pointer',
                  padding: '6px 0',
                  display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 4,
                  fontFamily: SANS,
                }}
              >
                <span style={{
                  fontSize: 11, fontWeight: 600, letterSpacing: 0.4,
                  color: sel ? T.amberDp : T.ink3,
                }}>
                  {RU_WEEKDAYS_SHORT[i].toUpperCase()}
                </span>
                <span style={{
                  width: 36, height: 36, borderRadius: '50%',
                  background: sel ? T.amberDp : 'transparent',
                  color: sel ? '#fff' : T.ink,
                  display: 'flex', alignItems: 'center', justifyContent: 'center',
                  fontFamily: SERIF, fontSize: 18,
                  border: !sel && isToday2 ? `1.5px solid ${T.amberDp}` : 'none',
                  fontVariantNumeric: 'tabular-nums',
                }}>{d.getDate()}</span>
                <div style={{
                  display: 'flex', gap: 2.5, height: 4, alignItems: 'center', marginTop: 2,
                }}>
                  {count > 0
                    ? Array.from({ length: Math.min(count, 3) }).map((_, k) => (
                        <span
                          key={k}
                          style={{
                            width: 4, height: 4, borderRadius: '50%',
                            background: sel ? T.amberDp : T.ink4,
                          }}
                        />
                      ))
                    : <span style={{ width: 4, height: 4 }} />}
                </div>
              </button>
            );
          })}
        </div>
      </div>

      {/* Day label */}
      <div style={{
        padding: '4px 20px 8px',
        display: 'flex', alignItems: 'baseline', justifyContent: 'space-between',
      }}>
        <div style={{ fontFamily: SERIF, fontSize: 22, color: T.ink, letterSpacing: -0.2 }}>
          {dateLabel}
        </div>
        <div style={{ fontSize: 12, color: T.ink3 }}>
          {pluralEvents(dayEvents.length)}
        </div>
      </div>

      {loading ? (
        <Placeholder text="Загрузка…" />
      ) : error ? (
        <ErrorState message={error} onRetry={() => void fetchEvents()} />
      ) : dayEvents.length === 0 ? (
        <EmptyDay />
      ) : (
        <div style={{ padding: '8px 16px 12px', display: 'grid', gap: 10 }}>
          {dayEvents.map((e) => (
            <AgendaRow
              key={e.id}
              event={e}
              isToday={isSelectedToday}
              onClick={() => setSelectedEvent(e)}
            />
          ))}
        </div>
      )}

      {selectedEvent && (
        <TaskDetailSheet
          key={selectedEvent.id}
          task={toDetailTask(selectedEvent)}
          onClose={() => setSelectedEvent(null)}
          onComplete={() => handleAction(selectedEvent.id, completeTask)}
          onPostpone={() => handleAction(selectedEvent.id, postponeTask)}
          onDelete={()   => handleAction(selectedEvent.id, deleteTask)}
          onSave={(patch) => handleSave(selectedEvent.id, patch)}
        />
      )}
    </>
  );
};

function toDetailTask(e: ScheduleEvent): DetailTask {
  return {
    id: e.id,
    title: e.title,
    description: e.raw.description,
    status: mapStatus(e.raw.status),
    priority: e.priority,
    category: e.category,
    startLabel: formatDateTime(e.startTime),
    endLabel: formatDateTime(e.endTime),
    startISO: e.raw.startTime,
    endISO:   e.raw.endTime,
  };
}

function formatDateTime(d: Date): string {
  const today = startOfDay(new Date());
  const tomorrow = new Date(today);
  tomorrow.setDate(today.getDate() + 1);
  const day = startOfDay(d);
  const time = toHHMM(d);
  if (day.getTime() === today.getTime())    return `Сегодня, ${time}`;
  if (day.getTime() === tomorrow.getTime()) return `Завтра, ${time}`;
  return `${d.getDate()} ${RU_MONTHS_GEN[d.getMonth()]}, ${time}`;
}

interface AgendaRowProps {
  event: ScheduleEvent;
  isToday: boolean;
  onClick: () => void;
}

function AgendaRow({ event, isToday, onClick }: AgendaRowProps) {
  const pal = T[event.priority];
  const catPal = T[event.category];
  const now = new Date();
  const isNow = isToday && now >= event.startTime && now <= event.endTime;

  return (
    <div style={{ display: 'flex', gap: 12, alignItems: 'stretch' }}>
      {/* Time column */}
      <div style={{
        width: 56, flexShrink: 0, paddingTop: 14,
        display: 'flex', flexDirection: 'column', alignItems: 'flex-end',
      }}>
        <div style={{
          fontFamily: SERIF, fontSize: 18, color: T.ink,
          letterSpacing: -0.2, fontVariantNumeric: 'tabular-nums', lineHeight: 1,
        }}>{toHHMM(event.startTime)}</div>
        <div style={{
          fontSize: 11, color: T.ink4, marginTop: 4, fontVariantNumeric: 'tabular-nums',
        }}>{toHHMM(event.endTime)}</div>
      </div>

      {/* Card */}
      <button
        type="button"
        onClick={onClick}
        style={{
          flex: 1, textAlign: 'left',
          background: T.surface,
          border: `0.5px solid ${T.hairline}`,
          borderRadius: 14,
          padding: '12px 14px 12px 16px',
          position: 'relative', overflow: 'hidden',
          fontFamily: SANS, cursor: 'pointer',
        }}
      >
        <div style={{
          position: 'absolute', left: 0, top: 12, bottom: 12,
          width: 3, borderRadius: '0 3px 3px 0', background: pal.rail,
        }} />
        <div style={{
          fontFamily: SERIF, fontSize: 17, color: T.ink, letterSpacing: -0.1,
          marginBottom: 6, lineHeight: 1.2,
        }}>{event.title}</div>
        <div style={{ display: 'flex', gap: 10, alignItems: 'center', flexWrap: 'wrap' }}>
          <span style={{
            display: 'inline-flex', alignItems: 'center', gap: 5,
            fontSize: 12, color: T.ink3,
          }}>
            <span style={{ width: 6, height: 6, borderRadius: '50%', background: catPal.rail }} />
            {CATEGORY_LABELS[event.category]}
          </span>
          {isNow && (
            <span style={{
              marginLeft: 'auto',
              fontSize: 11, fontWeight: 600,
              background: T.now, color: '#fff',
              padding: '2px 8px', borderRadius: 999,
              letterSpacing: 0.3,
            }}>СЕЙЧАС</span>
          )}
        </div>
      </button>
    </div>
  );
}

function Placeholder({ text }: { text: string }) {
  return (
    <div style={{
      padding: '64px 16px', textAlign: 'center',
      color: T.ink3, fontSize: 14,
    }}>{text}</div>
  );
}


function EmptyDay() {
  return (
    <div style={{ padding: '48px 32px 16px', textAlign: 'center' }}>
      <div style={{
        width: 56, height: 56, margin: '0 auto 14px',
        borderRadius: '50%', background: T.amberFill,
        display: 'flex', alignItems: 'center', justifyContent: 'center',
        color: T.amberDp,
      }}>
        <CalendarOff size={26} strokeWidth={1.6} />
      </div>
      <div style={{
        fontFamily: SERIF, fontSize: 22, color: T.ink,
        letterSpacing: -0.2, marginBottom: 6,
      }}>Свободный день</div>
      <div style={{
        fontSize: 13.5, color: T.ink3, lineHeight: 1.45,
        maxWidth: 280, margin: '0 auto',
      }}>На выбранный день событий нет</div>
    </div>
  );
}

export default ScheduleScreen;
