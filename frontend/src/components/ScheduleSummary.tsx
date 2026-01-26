import { Clock, CheckCircle, Calendar, TrendingUp, AlertTriangle } from 'lucide-react';
import { ScheduleEvent, ViewMode } from '../screens/ScheduleScreen';

interface ScheduleSummaryProps {
  events: ScheduleEvent[];
  currentDate: Date;
  viewMode: ViewMode;
}

const ScheduleSummary = ({ events, currentDate, viewMode }: ScheduleSummaryProps) => {
  const getTodayEvents = () => {
    return events.filter((event) => {
      const eventDate = new Date(event.date);
      return eventDate.toDateString() === currentDate.toDateString();
    });
  };

  const calculateFocusTime = () => {
    const todayEvents = getTodayEvents();
    let totalMinutes = 0;

    todayEvents.forEach((event) => {
      const [startHour, startMinute] = event.startTime.split(':').map(Number);
      const [endHour, endMinute] = event.endTime.split(':').map(Number);
      const duration = (endHour * 60 + endMinute) - (startHour * 60 + startMinute);
      totalMinutes += duration;
    });

    const hours = Math.floor(totalMinutes / 60);
    const minutes = totalMinutes % 60;
    return `${hours}ч ${minutes}м`;
  };

  const getFreeSlots = () => {
    const todayEvents = getTodayEvents().sort((a, b) => {
      const aTime = parseInt(a.startTime.replace(':', ''));
      const bTime = parseInt(b.startTime.replace(':', ''));
      return aTime - bTime;
    });

    const slots = [];
    let lastEndTime = '08:00';

    todayEvents.forEach((event) => {
      if (event.startTime > lastEndTime) {
        slots.push({
          start: lastEndTime,
          end: event.startTime,
        });
      }
      lastEndTime = event.endTime;
    });

    if (lastEndTime < '22:00') {
      slots.push({
        start: lastEndTime,
        end: '22:00',
      });
    }

    return slots;
  };

  const todayEvents = getTodayEvents();
  const freeSlots = getFreeSlots();

  return (
    <div className="h-full flex flex-col">
      <div className="p-6 border-b border-gray-200">
        <h3 className="font-semibold text-lg text-[#2D2F31] mb-1">
          {viewMode === 'day' ? 'Резюме дня' : 'Обзор недели'}
        </h3>
        <p className="text-sm text-gray-500">
          {currentDate.toLocaleDateString('ru-RU', {
            day: 'numeric',
            month: 'long'
          })}
        </p>
      </div>

      <div className="flex-1 overflow-auto p-6">
        <div className="grid grid-cols-2 gap-3 mb-6">
          <div className="bg-gradient-to-br from-[#5C6BFF] to-[#4C5BEF] rounded-xl p-4 text-white">
            <div className="flex items-center gap-2 mb-2">
              <Calendar size={16} />
              <span className="text-xs font-medium opacity-90">Задач сегодня</span>
            </div>
            <p className="text-2xl font-bold">{todayEvents.length}</p>
          </div>

          <div className="bg-gradient-to-br from-[#4CB782] to-[#3CA672] rounded-xl p-4 text-white">
            <div className="flex items-center gap-2 mb-2">
              <Clock size={16} />
              <span className="text-xs font-medium opacity-90">Время фокуса</span>
            </div>
            <p className="text-2xl font-bold">{calculateFocusTime()}</p>
          </div>
        </div>

        <div className="mb-6">
          <div className="flex items-center gap-2 mb-3">
            <CheckCircle size={18} className="text-[#4CB782]" />
            <h4 className="font-semibold text-[#2D2F31]">События сегодня</h4>
          </div>
          <div className="space-y-2">
            {todayEvents.length > 0 ? (
              todayEvents.map((event) => (
                <div
                  key={event.id}
                  className="p-3 bg-[#F7F8FA] rounded-lg border border-gray-200"
                >
                  <div className="flex items-start justify-between mb-1">
                    <p className="font-medium text-sm text-[#2D2F31] flex-1">{event.title}</p>
                    <span
                      className={`w-2 h-2 rounded-full mt-1.5 ${
                        event.priority === 'high'
                          ? 'bg-[#FF8A65]'
                          : event.priority === 'medium'
                          ? 'bg-[#5C6BFF]'
                          : 'bg-[#4CB782]'
                      }`}
                    />
                  </div>
                  <p className="text-xs text-gray-500">
                    {event.startTime} - {event.endTime}
                  </p>
                </div>
              ))
            ) : (
              <p className="text-sm text-gray-500 text-center py-4">Нет событий на сегодня</p>
            )}
          </div>
        </div>

        <div className="mb-6">
          <div className="flex items-center gap-2 mb-3">
            <TrendingUp size={18} className="text-[#5C6BFF]" />
            <h4 className="font-semibold text-[#2D2F31]">Свободные окна</h4>
          </div>
          <div className="space-y-2">
            {freeSlots.length > 0 ? (
              freeSlots.map((slot, index) => (
                <div
                  key={index}
                  className="p-3 bg-[#4CB782]/10 rounded-lg border border-[#4CB782]/20"
                >
                  <p className="text-sm text-[#2D2F31] font-medium">
                    {slot.start} - {slot.end}
                  </p>
                </div>
              ))
            ) : (
              <p className="text-sm text-gray-500 text-center py-4">Нет свободных окон</p>
            )}
          </div>
        </div>

        <div className="p-4 bg-gradient-to-br from-[#FF8A65]/10 to-[#FF8A65]/5 rounded-xl border border-[#FF8A65]/20">
          <div className="flex items-start gap-3">
            <div className="w-8 h-8 bg-[#FF8A65] rounded-lg flex items-center justify-center flex-shrink-0">
              <AlertTriangle size={16} className="text-white" />
            </div>
            <div>
              <h4 className="font-semibold text-[#2D2F31] mb-1 text-sm">Рекомендация</h4>
              <p className="text-xs text-gray-600 leading-relaxed">
                Лучше сдвинуть встречу с клиентом на утро — вы более продуктивны в это время
              </p>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};

export default ScheduleSummary;
