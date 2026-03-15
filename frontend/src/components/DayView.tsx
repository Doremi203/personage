import { ScheduleEvent } from '../screens/ScheduleScreen';
import { toYYYYMMDD } from '../utils/dateUtils';

interface DayViewProps {
  events: ScheduleEvent[];
  currentDate: Date;
}

const DayView = ({ events, currentDate }: DayViewProps) => {
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

    const top = ((startHour - 8) * 60 + startMinute) * (80 / 60);
    const duration = (endHour * 60 + endMinute) - (startHour * 60 + startMinute);
    const height = Math.max(duration * (80 / 60), 20);

    return { top, height };
  };

  const todayEvents = events.filter((event) => {
    // Compare YYYY-MM-DD strings to avoid UTC-vs-local day shift
    return event.date === toYYYYMMDD(currentDate);
  });

  return (
    <div className="bg-white rounded-2xl border border-gray-200 overflow-hidden">
      <div className="relative">
        <div className="flex">
          <div className="w-24">
            {hours.map((hour) => (
              <div key={hour} className="h-20 p-3 text-sm text-gray-500 text-right border-b border-gray-100">
                {hour.toString().padStart(2, '0')}:00
              </div>
            ))}
          </div>

          <div className="flex-1 relative border-l border-gray-200">
            {hours.map((hour) => (
              <div
                key={hour}
                className="h-20 border-b border-gray-100 hover:bg-gray-50/50 transition-colors cursor-pointer"
              />
            ))}

            {todayEvents.map((event) => {
              const { top, height } = getEventPosition(event);
              return (
                <div
                  key={event.id}
                  className={`absolute left-2 right-2 rounded-xl border-l-4 p-4 text-white shadow-lg cursor-pointer hover:shadow-xl transition-shadow ${getPriorityColor(
                    event.priority
                  )}`}
                  style={{ top: `${top}px`, height: `${height}px` }}
                >
                  <div className="font-semibold mb-1">{event.title}</div>
                  <div className="text-sm opacity-90">
                    {event.startTime} - {event.endTime}
                  </div>
                </div>
              );
            })}
          </div>
        </div>
      </div>
    </div>
  );
};

export default DayView;
