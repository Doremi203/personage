import { useState, useEffect, useCallback } from 'react';
import { ChevronLeft, ChevronRight, Calendar as CalendarIcon, Loader2 } from 'lucide-react';
import WeekView from '../components/WeekView';
import DayView from '../components/DayView';
import ScheduleSummary from '../components/ScheduleSummary';
import {
  listTasks,
  ApiTaskItem,
  ApiTaskPriority,
  ApiTaskCategory,
} from '../utils/taskerService';

export type ViewMode = 'week' | 'day';

export interface ScheduleEvent {
  id: string;
  title: string;
  startTime: string;
  endTime: string;
  date: string;
  priority: 'high' | 'medium' | 'low';
  category: string;
}

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

function mapApiTaskToEvent(task: ApiTaskItem): ScheduleEvent | null {
  if (!task.startTime) return null;
  const startHHMM = toHHMM(task.startTime);
  const endHHMM = task.endTime ? toHHMM(task.endTime) : startHHMM;
  const date = new Date(task.startTime).toISOString().split('T')[0];
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

const ScheduleScreen = () => {
  const [viewMode, setViewMode] = useState<ViewMode>('week');
  const [currentDate, setCurrentDate] = useState(new Date());
  const [events, setEvents] = useState<ScheduleEvent[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchEvents = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      let from: Date;
      let till: Date;
      if (viewMode === 'week') {
        from = new Date(currentDate);
        // (getDay() + 6) % 7 maps Sun=0→6, Mon=1→0, ..., Sat=6→5, giving days since Monday
        from.setDate(currentDate.getDate() - ((currentDate.getDay() + 6) % 7));
        till = new Date(from);
        till.setDate(from.getDate() + 6);
      } else {
        from = new Date(currentDate);
        till = new Date(currentDate);
      }
      const response = await listTasks({
        from: toApiDateParam(from),
        till: toApiDateParam(till),
        pageSize: 100,
        page: 1,
      });
      const mapped = (response.tasks ?? [])
        .map(mapApiTaskToEvent)
        .filter((e): e is ScheduleEvent => e !== null);
      setEvents(mapped);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Не удалось загрузить расписание');
    } finally {
      setLoading(false);
    }
  }, [currentDate, viewMode]);

  useEffect(() => {
    void fetchEvents();
  }, [fetchEvents]);

  const navigateDate = (direction: 'prev' | 'next') => {
    const newDate = new Date(currentDate);
    if (viewMode === 'week') {
      newDate.setDate(currentDate.getDate() + (direction === 'next' ? 7 : -7));
    } else {
      newDate.setDate(currentDate.getDate() + (direction === 'next' ? 1 : -1));
    }
    setCurrentDate(newDate);
  };

  const formatDateRange = () => {
    if (viewMode === 'week') {
      const weekStart = new Date(currentDate);
      weekStart.setDate(currentDate.getDate() - ((currentDate.getDay() + 6) % 7));
      const weekEnd = new Date(weekStart);
      weekEnd.setDate(weekStart.getDate() + 6);

      return `${weekStart.getDate()} ${weekStart.toLocaleDateString('ru-RU', { month: 'short' })} - ${weekEnd.getDate()} ${weekEnd.toLocaleDateString('ru-RU', { month: 'short' })} ${weekEnd.getFullYear()}`;
    } else {
      return currentDate.toLocaleDateString('ru-RU', {
        day: 'numeric',
        month: 'long',
        year: 'numeric'
      });
    }
  };

  return (
    <div className="h-full flex flex-col md:flex-row md:pt-0 pt-16">
      <div className="flex-1 flex flex-col">
        <div className="p-4 md:p-8 border-b border-gray-200">
          <div className="flex flex-col md:flex-row md:items-center md:justify-between gap-4 mb-4 md:mb-6">
            <h2 className="text-xl md:text-2xl font-bold text-[#2D2F31]">Расписание</h2>
            <div className="flex items-center gap-2">
              <button
                onClick={() => setViewMode('week')}
                className={`px-3 md:px-4 py-2 rounded-lg font-medium transition-colors text-sm md:text-base ${
                  viewMode === 'week'
                    ? 'bg-[#5C6BFF] text-white'
                    : 'bg-gray-100 text-gray-700 hover:bg-gray-200'
                }`}
              >
                Неделя
              </button>
              <button
                onClick={() => setViewMode('day')}
                className={`px-3 md:px-4 py-2 rounded-lg font-medium transition-colors text-sm md:text-base ${
                  viewMode === 'day'
                    ? 'bg-[#5C6BFF] text-white'
                    : 'bg-gray-100 text-gray-700 hover:bg-gray-200'
                }`}
              >
                День
              </button>
            </div>
          </div>

          <div className="flex flex-col md:flex-row md:items-center md:justify-between gap-3">
            <div className="flex items-center gap-2 md:gap-4 overflow-x-auto">
              <button
                onClick={() => navigateDate('prev')}
                className="p-2 hover:bg-gray-100 rounded-lg transition-colors flex-shrink-0"
              >
                <ChevronLeft size={20} className="text-gray-600" />
              </button>
              <div className="flex items-center gap-2 flex-shrink-0">
                <CalendarIcon size={18} className="text-[#5C6BFF]" />
                <span className="font-semibold text-[#2D2F31] text-sm md:text-base">{formatDateRange()}</span>
              </div>
              <button
                onClick={() => navigateDate('next')}
                className="p-2 hover:bg-gray-100 rounded-lg transition-colors flex-shrink-0"
              >
                <ChevronRight size={20} className="text-gray-600" />
              </button>
            </div>
            <button
              onClick={() => setCurrentDate(new Date())}
              className="px-4 py-2 bg-gray-100 text-gray-700 rounded-lg hover:bg-gray-200 transition-colors font-medium text-sm"
            >
              Сегодня
            </button>
          </div>
        </div>

        <div className="flex-1 overflow-auto p-4 md:p-8">
          {loading ? (
            <div className="flex items-center justify-center py-16 text-gray-400">
              <Loader2 size={32} className="animate-spin" />
            </div>
          ) : error ? (
            <div className="py-8 text-center">
              <p className="text-red-500 text-sm mb-3">{error}</p>
              <button
                onClick={() => void fetchEvents()}
                className="px-4 py-2 bg-[#5C6BFF] text-white rounded-xl text-sm hover:bg-[#4C5BEF] transition-colors"
              >
                Повторить
              </button>
            </div>
          ) : viewMode === 'week' ? (
            <WeekView events={events} currentDate={currentDate} />
          ) : (
            <DayView events={events} currentDate={currentDate} />
          )}
        </div>
      </div>

      <div className="hidden md:block md:w-96 bg-white border-l border-gray-200">
        <ScheduleSummary events={events} currentDate={currentDate} viewMode={viewMode} />
      </div>
    </div>
  );
};

export default ScheduleScreen;
