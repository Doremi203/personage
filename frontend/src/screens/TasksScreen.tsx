import { useCallback, useEffect, useMemo, useState } from 'react';
import {
  CalendarClock,
  CheckCheck,
  Calendar as CalendarIcon,
  Inbox,
  RefreshCw,
  Sun,
  type LucideIcon,
} from 'lucide-react';
import {
  LargeHeader,
  SearchBar,
  Segmented,
  type SegmentedItem,
} from '../mobile/Chrome';
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
  ApiTaskStatusFilter,
  completeTask,
  deleteTask,
  listTasks,
  postponeTask,
  updateTask,
  type ApiTaskItem,
  type UpdateTaskPatch,
} from '../utils/taskerService';
import { RU_MONTHS_GEN, startOfDay, toApiDateParam } from '../utils/dateFormat';

type Filter = 'today' | 'upcoming' | 'inbox' | 'done';

interface Task {
  id: string;
  title: string;
  description: string;
  status: 'unplanned' | 'planned' | 'completed';
  priority: Priority;
  category: Category;
  deadline: string;        // pre-formatted Russian display string
  deadlineDate?: Date;     // raw Date for sorting
  raw: ApiTaskItem;        // kept for the detail sheet
}

function formatDeadline(iso: string | undefined): string {
  if (!iso) return 'Без даты';
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return 'Без даты';
  const today = startOfDay(new Date());
  const tomorrow = new Date(today);
  tomorrow.setDate(today.getDate() + 1);
  const day = startOfDay(d);
  const hh = String(d.getHours()).padStart(2, '0');
  const mm = String(d.getMinutes()).padStart(2, '0');
  const time = `${hh}:${mm}`;
  if (day.getTime() === today.getTime())    return `Сегодня, ${time}`;
  if (day.getTime() === tomorrow.getTime()) return `Завтра, ${time}`;
  const sameYear = d.getFullYear() === today.getFullYear();
  return `${d.getDate()} ${RU_MONTHS_GEN[d.getMonth()]}${sameYear ? '' : ' ' + d.getFullYear()}`;
}

function mapStatus(status: string): Task['status'] {
  if (status === ApiTaskStatus.COMPLETED) return 'completed';
  if (status === ApiTaskStatus.UNPLANNED) return 'unplanned';
  return 'planned';
}

function mapPriority(priority: string): Priority {
  if (priority === ApiTaskPriority.HIGH) return 'high';
  if (priority === ApiTaskPriority.LOW)  return 'low';
  return 'medium';
}

function mapCategory(category: string): Category {
  if (category === ApiTaskCategory.WORK)  return 'work';
  if (category === ApiTaskCategory.STUDY) return 'study';
  return 'personal';
}

function mapApiTask(t: ApiTaskItem): Task {
  const deadlineSource = t.deadline ?? t.startTime;
  const dl = deadlineSource ? new Date(deadlineSource) : undefined;
  return {
    id: t.id,
    title: t.title,
    description: t.description,
    status: mapStatus(t.status),
    priority: mapPriority(t.priority),
    category: mapCategory(t.category),
    deadline: formatDeadline(deadlineSource),
    deadlineDate: dl && !Number.isNaN(dl.getTime()) ? dl : undefined,
    raw: t,
  };
}

function toDetailTask(task: Task): DetailTask {
  const start = task.raw.startTime ?? task.raw.deadline;
  const end   = task.raw.endTime   ?? task.raw.deadline;
  return {
    id: task.id,
    title: task.title,
    description: task.description,
    status: task.status,
    priority: task.priority,
    category: task.category,
    startLabel: start ? formatDeadline(start) : '—',
    endLabel:   end   ? formatDeadline(end)   : '—',
    startISO: task.raw.startTime,
    endISO:   task.raw.endTime,
  };
}

interface FilterParams {
  status?: string;
  from?: string;
  till?: string;
}

function paramsForFilter(filter: Filter): FilterParams {
  const today = new Date();
  const todayParam = toApiDateParam(today);
  const tomorrow = new Date(today);
  tomorrow.setDate(today.getDate() + 1);
  switch (filter) {
    case 'today':
      return { status: ApiTaskStatusFilter.PLANNED, from: todayParam, till: todayParam };
    case 'upcoming':
      return { status: ApiTaskStatusFilter.PLANNED, from: toApiDateParam(tomorrow) };
    case 'inbox':
      return { status: ApiTaskStatusFilter.UNPLANNED };
    case 'done':
      return { status: ApiTaskStatusFilter.COMPLETED };
  }
}

const PAGE_SIZE = 50;

const TasksScreen = () => {
  const [filter, setFilter] = useState<Filter>('today');
  const [search, setSearch] = useState('');
  const [debouncedSearch, setDebouncedSearch] = useState('');
  const [tasks, setTasks] = useState<Task[]>([]);
  const [counts, setCounts] = useState<Record<Filter, number>>({
    today: 0, upcoming: 0, inbox: 0, done: 0,
  });
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [selectedTask, setSelectedTask] = useState<Task | null>(null);

  useEffect(() => {
    const t = setTimeout(() => setDebouncedSearch(search), 300);
    return () => clearTimeout(t);
  }, [search]);

  const fetchAll = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const [list, c1, c2, c3, c4] = await Promise.all([
        listTasks({
          ...paramsForFilter(filter),
          text: debouncedSearch || undefined,
          pageSize: PAGE_SIZE,
          page: 1,
        }),
        listTasks({ ...paramsForFilter('today'),    pageSize: 1, page: 1 }),
        listTasks({ ...paramsForFilter('upcoming'), pageSize: 1, page: 1 }),
        listTasks({ ...paramsForFilter('inbox'),    pageSize: 1, page: 1 }),
        listTasks({ ...paramsForFilter('done'),     pageSize: 1, page: 1 }),
      ]);
      setTasks((list.tasks ?? []).map(mapApiTask));
      setCounts({
        today:    c1.total ?? 0,
        upcoming: c2.total ?? 0,
        inbox:    c3.total ?? 0,
        done:     c4.total ?? 0,
      });
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Не удалось загрузить задачи');
    } finally {
      setLoading(false);
    }
  }, [filter, debouncedSearch]);

  useEffect(() => {
    void fetchAll();
  }, [fetchAll]);

  const runAction = useCallback(
    async (id: string, action: (id: string) => Promise<unknown>) => {
      await action(id);
      setSelectedTask(null);
      await fetchAll();
    },
    [fetchAll],
  );

  const handleSave = useCallback(
    async (id: string, patch: UpdateTaskPatch) => {
      const updated = await updateTask(id, patch);
      setSelectedTask(mapApiTask(updated));
      await fetchAll();
    },
    [fetchAll],
  );

  const items: SegmentedItem<Filter>[] = useMemo(() => [
    { id: 'today',    label: 'Сегодня',  count: counts.today },
    { id: 'upcoming', label: 'Скоро',    count: counts.upcoming },
    { id: 'inbox',    label: 'Без даты', count: counts.inbox },
    { id: 'done',     label: 'Готово',   count: counts.done },
  ], [counts]);

  const subtitle =
    counts.today > 0 ? `${counts.today} запланировано на сегодня`
                     : 'На сегодня всё готово';

  return (
    <>
      <LargeHeader title="Задачи" subtitle={subtitle} />

      <SearchBar
        value={search}
        onChange={setSearch}
        placeholder="Найти задачу"
      />

      <Segmented value={filter} onChange={setFilter} items={items} />

      {loading ? (
        <ListPlaceholder text="Загрузка…" />
      ) : error ? (
        <ErrorState message={error} onRetry={() => void fetchAll()} />
      ) : tasks.length === 0 ? (
        <EmptyState filter={filter} />
      ) : (
        <div style={{ padding: '0 16px', display: 'grid', gap: 10 }}>
          {tasks.map((task) => (
            <TaskListCard
              key={task.id}
              task={task}
              onClick={() => setSelectedTask(task)}
            />
          ))}
        </div>
      )}

      <SyncFooter />

      {selectedTask && (
        <TaskDetailSheet
          key={selectedTask.id}
          task={toDetailTask(selectedTask)}
          onClose={() => setSelectedTask(null)}
          onComplete={() => runAction(selectedTask.id, completeTask)}
          onPostpone={() => runAction(selectedTask.id, postponeTask)}
          onDelete={()   => runAction(selectedTask.id, deleteTask)}
          onSave={(patch) => handleSave(selectedTask.id, patch)}
        />
      )}
    </>
  );
};

interface TaskListCardProps {
  task: Task;
  onClick: () => void;
}

function TaskListCard({ task, onClick }: TaskListCardProps) {
  const pal = T[task.priority];
  const catPal = T[task.category];
  const isDone = task.status === 'completed';

  return (
    <button
      type="button"
      onClick={onClick}
      style={{
        width: '100%', textAlign: 'left',
        background: T.surface,
        border: `0.5px solid ${T.hairline}`,
        borderRadius: 14,
        padding: '14px 14px 14px 16px',
        cursor: 'pointer',
        position: 'relative',
        fontFamily: SANS,
        display: 'flex', flexDirection: 'column', gap: 10,
      }}
    >
      <div style={{
        position: 'absolute', left: 0, top: 14, bottom: 14, width: 3,
        borderRadius: '0 3px 3px 0',
        background: isDone ? T.subtleHi : pal.rail,
      }} />

      <div style={{ display: 'flex', gap: 12, alignItems: 'flex-start' }}>
        <div style={{
          width: 22, height: 22, borderRadius: '50%',
          border: `1.5px solid ${isDone ? T.ok : T.ink4}`,
          background: isDone ? T.ok : 'transparent',
          display: 'flex', alignItems: 'center', justifyContent: 'center',
          flexShrink: 0, marginTop: 1,
        }}>
          {isDone && (
            <svg width="11" height="9" viewBox="0 0 11 9" aria-hidden>
              <path d="M1 4.5L4 7.5L10 1.5" stroke="#fff" strokeWidth="2"
                strokeLinecap="round" strokeLinejoin="round" fill="none" />
            </svg>
          )}
        </div>

        <div style={{ flex: 1, minWidth: 0 }}>
          <div style={{
            fontFamily: SERIF, fontSize: 18, lineHeight: 1.2,
            color: T.ink, letterSpacing: -0.1,
            textDecoration: isDone ? 'line-through' : 'none',
            textDecorationColor: T.ink4,
            marginBottom: 3,
            wordBreak: 'break-word',
          }}>{task.title}</div>
          {task.description && (
            <div style={{
              fontSize: 13, color: T.ink3, lineHeight: 1.4,
              display: '-webkit-box', WebkitLineClamp: 1, WebkitBoxOrient: 'vertical',
              overflow: 'hidden',
            }}>{task.description}</div>
          )}
        </div>
      </div>

      <div style={{
        display: 'flex', alignItems: 'center', gap: 10, paddingLeft: 34,
        flexWrap: 'wrap',
      }}>
        <span style={{
          display: 'inline-flex', alignItems: 'center', gap: 5,
          fontSize: 12, color: T.ink3, fontWeight: 500,
        }}>
          <CalendarIcon size={12} strokeWidth={1.8} />
          {task.deadline}
        </span>
        <span style={{
          display: 'inline-flex', alignItems: 'center', gap: 5,
          fontSize: 12, color: T.ink3, fontWeight: 500,
        }}>
          <span style={{ width: 6, height: 6, borderRadius: '50%', background: catPal.rail }} />
          {CATEGORY_LABELS[task.category]}
        </span>
      </div>
    </button>
  );
}

interface EmptyMsg {
  icon: LucideIcon;
  title: string;
  sub: string;
}

const EMPTY_BY_FILTER: Record<Filter, EmptyMsg> = {
  today:    { icon: Sun,           title: 'На сегодня свободно',     sub: 'Новые задачи появятся автоматически из почты, календаря и Telegram' },
  upcoming: { icon: CalendarClock, title: 'Нет предстоящих',         sub: 'Личный ассистент пока не нашёл новых задач на ближайшие дни' },
  inbox:    { icon: Inbox,         title: 'Входящие пусты',          sub: 'Задачи без даты будут появляться здесь' },
  done:     { icon: CheckCheck,    title: 'Пока ничего не сделано',  sub: 'Завершённые задачи появятся здесь' },
};

function EmptyState({ filter }: { filter: Filter }) {
  const msg = EMPTY_BY_FILTER[filter];
  const Icon = msg.icon;
  return (
    <div style={{ padding: '48px 32px 16px', textAlign: 'center' }}>
      <div style={{
        width: 56, height: 56, margin: '0 auto 14px',
        borderRadius: '50%', background: T.amberFill,
        display: 'flex', alignItems: 'center', justifyContent: 'center',
        color: T.amberDp,
      }}>
        <Icon size={26} strokeWidth={1.6} />
      </div>
      <div style={{
        fontFamily: SERIF, fontSize: 22, color: T.ink,
        letterSpacing: -0.2, marginBottom: 6,
      }}>{msg.title}</div>
      <div style={{
        fontSize: 13.5, color: T.ink3, lineHeight: 1.45,
        maxWidth: 280, margin: '0 auto',
      }}>{msg.sub}</div>
    </div>
  );
}

function ListPlaceholder({ text }: { text: string }) {
  return (
    <div style={{
      padding: '64px 16px', textAlign: 'center',
      color: T.ink3, fontSize: 14,
    }}>{text}</div>
  );
}

function SyncFooter() {
  return (
    <div style={{
      padding: '20px 16px 8px',
      display: 'flex', alignItems: 'center', justifyContent: 'center', gap: 6,
      color: T.ink4, fontSize: 11.5,
    }}>
      <RefreshCw size={11} strokeWidth={1.8} />
      Обновлено только что
    </div>
  );
}

export default TasksScreen;
