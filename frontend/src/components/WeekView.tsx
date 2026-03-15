import { ScheduleEvent } from '../screens/ScheduleScreen';
import { toYYYYMMDD, layoutEvents, SLOT_HEIGHT_PX } from '../utils/dateUtils';

interface WeekViewProps {
  events: ScheduleEvent[];
  currentDate: Date;
}

const WeekView = ({ events, currentDate }: WeekViewProps) => {
  const getWeekDays = () => {
    const days = [];
    const startOfWeek = new Date(currentDate);
    // (getDay() + 6) % 7 → Mon=0 … Sun=6, so we always land on Monday
    startOfWeek.setDate(currentDate.getDate() - ((currentDate.getDay() + 6) % 7));

    for (let i = 0; i < 7; i++) {
      const day = new Date(startOfWeek);
      day.setDate(startOfWeek.getDate() + i);
      days.push(day);
    }
    return days;
  };

  const weekDays = getWeekDays();
  const hours = Array.from({ length: 14 }, (_, i) => i + 8);

  const getPriorityColor = (priority: string) => {
    switch (priority) {
      case 'high':
        return 'bg-[#FF8A65] border-[#FF7A55]';
      case 'medium':
        return 'bg-[#5C6BFF] border-[#4C5BEF]';
      case 'low':
        return 'bg-[#4CB782] border-[#3CA672]';
      default:
        return 'bg-gray-400 border-gray-500';
    }
  };

  const getEventPosition = (event: ScheduleEvent) => {
    const [startHour, startMinute] = event.startTime.split(':').map(Number);
    const [endHour, endMinute] = event.endTime.split(':').map(Number);

    const top = ((startHour - 8) * 60 + startMinute) * (SLOT_HEIGHT_PX / 60);
    const duration = (endHour * 60 + endMinute) - (startHour * 60 + startMinute);
    const height = Math.max(duration * (SLOT_HEIGHT_PX / 60), 16);

    return { top, height };
  };

  const isToday = (date: Date) => {
    const today = new Date();
    return date.toDateString() === today.toDateString();
  };

  return (
    <div className="bg-white rounded-2xl border border-gray-200 overflow-hidden">
      {/* Header row */}
      <div className="flex border-b border-gray-200">
        <div className="w-20 flex-shrink-0 p-4" />
        {weekDays.map((day, index) => (
          <div
            key={index}
            className={`flex-1 p-4 text-center border-l border-gray-200 ${
              isToday(day) ? 'bg-[#5C6BFF]/5' : ''
            }`}
          >
            <div className="text-xs font-medium text-gray-500 mb-1">
              {day.toLocaleDateString('ru-RU', { weekday: 'short' })}
            </div>
            <div
              className={`text-lg font-semibold ${
                isToday(day) ? 'text-[#5C6BFF]' : 'text-[#2D2F31]'
              }`}
            >
              {day.getDate()}
            </div>
          </div>
        ))}
      </div>

      {/* Time grid */}
      <div className="flex">
        {/* Hour labels */}
        <div className="w-20 flex-shrink-0">
          {hours.map((hour) => (
            <div key={hour} className="h-20 p-2 text-xs text-gray-500 text-right border-b border-gray-100">
              {hour.toString().padStart(2, '0')}:00
            </div>
          ))}
        </div>

        {/* Day columns */}
        {weekDays.map((day, dayIndex) => {
          const dayEvents = events.filter(
            (event) => event.date === toYYYYMMDD(day)
          );
          const layouts = layoutEvents(dayEvents);
          return (
            <div key={dayIndex} className="flex-1 relative border-l border-gray-200 overflow-hidden">
              {hours.map((hour) => (
                <div
                  key={hour}
                  className="h-20 border-b border-gray-100 hover:bg-gray-50/50 transition-colors cursor-pointer"
                />
              ))}

              {layouts.map(({ id, col, totalCols }) => {
                const event = dayEvents.find((e) => e.id === id)!;
                const { top, height } = getEventPosition(event);
                const pct = 100 / totalCols;
                return (
                  <div
                    key={event.id}
                    className={`absolute rounded-lg border-l-4 p-1 text-white shadow-lg cursor-pointer hover:shadow-xl transition-shadow overflow-hidden ${getPriorityColor(
                      event.priority
                    )}`}
                    style={{
                      top: `${top}px`,
                      height: `${height}px`,
                      left: `calc(${col * pct}% + 1px)`,
                      width: `calc(${pct}% - 2px)`,
                    }}
                  >
                    <div className="text-xs font-semibold line-clamp-1">{event.title}</div>
                    <div className="text-xs opacity-90">
                      {event.startTime}–{event.endTime}
                    </div>
                  </div>
                );
              })}
            </div>
          );
        })}
      </div>
    </div>
  );
};

export default WeekView;
