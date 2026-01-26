import { useState } from 'react';
import { ChevronLeft, ChevronRight, Calendar as CalendarIcon } from 'lucide-react';
import WeekView from '../components/WeekView';
import DayView from '../components/DayView';
import ScheduleSummary from '../components/ScheduleSummary';

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

const ScheduleScreen = () => {
  const [viewMode, setViewMode] = useState<ViewMode>('week');
  const [currentDate, setCurrentDate] = useState(new Date());

  const mockEvents: ScheduleEvent[] = [
    {
      id: '1',
      title: 'Встреча с клиентом',
      startTime: '09:00',
      endTime: '10:30',
      date: '2025-11-16',
      priority: 'high',
      category: 'work',
    },
    {
      id: '2',
      title: 'Разработка проекта',
      startTime: '11:00',
      endTime: '13:00',
      date: '2025-11-16',
      priority: 'medium',
      category: 'work',
    },
    {
      id: '3',
      title: 'Обед',
      startTime: '13:00',
      endTime: '14:00',
      date: '2025-11-16',
      priority: 'low',
      category: 'personal',
    },
    {
      id: '4',
      title: 'Тренировка',
      startTime: '18:00',
      endTime: '19:30',
      date: '2025-11-16',
      priority: 'medium',
      category: 'health',
    },
    {
      id: '5',
      title: 'Онлайн курс',
      startTime: '10:00',
      endTime: '12:00',
      date: '2025-11-17',
      priority: 'medium',
      category: 'study',
    },
  ];

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
      weekStart.setDate(currentDate.getDate() - currentDate.getDay() + 1);
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
          {viewMode === 'week' ? (
            <WeekView events={mockEvents} currentDate={currentDate} />
          ) : (
            <DayView events={mockEvents} currentDate={currentDate} />
          )}
        </div>
      </div>

      <div className="hidden md:block md:w-96 bg-white border-l border-gray-200">
        <ScheduleSummary events={mockEvents} currentDate={currentDate} viewMode={viewMode} />
      </div>
    </div>
  );
};

export default ScheduleScreen;
