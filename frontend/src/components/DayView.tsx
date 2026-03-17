import { ScheduleEvent } from '../screens/ScheduleScreen';
import { toYYYYMMDD, toMinutes, layoutEvents, SLOT_HEIGHT_PX, snapStart, snapEnd } from '../utils/dateUtils';

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
    const rawStart = toMinutes(event.startTime);
    const rawEnd = toMinutes(event.endTime);
    const snappedStart = snapStart(rawStart);
    const snappedEnd = snapEnd(rawEnd, snappedStart);

    const top = (snappedStart - 8 * 60) * (SLOT_HEIGHT_PX / 60);
    const height = (snappedEnd - snappedStart) * (SLOT_HEIGHT_PX / 60);

    return { top, height };
  };

  const todayEvents = events.filter((event) => {
    // Compare YYYY-MM-DD strings to avoid UTC-vs-local day shift
    return event.date === toYYYYMMDD(currentDate);
  });

  const layouts = layoutEvents(todayEvents);

  return (
    <div className="bg-white rounded-2xl border border-gray-200 overflow-hidden">
      <div className="relative">
        <div className="flex">
          <div className="w-24 flex-shrink-0">
            {hours.map((hour) => (
              <div key={hour} className="h-20 p-3 text-sm text-gray-500 text-right border-b border-gray-100">
                {hour.toString().padStart(2, '0')}:00
              </div>
            ))}
          </div>

          <div className="flex-1 relative border-l border-gray-200 overflow-hidden">
            {hours.map((hour) => (
              <div
                key={hour}
                className="h-20 border-b border-gray-100 hover:bg-gray-50/50 transition-colors cursor-pointer"
              />
            ))}

            {layouts.map(({ id, col, totalCols }) => {
              const event = todayEvents.find((e) => e.id === id)!;
              const { top, height } = getEventPosition(event);
              const pct = 100 / totalCols;
              return (
                <div
                  key={event.id}
                  className={`absolute rounded-xl border-l-4 p-2 text-white shadow-lg cursor-pointer hover:shadow-xl transition-shadow overflow-hidden ${getPriorityColor(
                    event.priority
                  )}`}
                  style={{
                    top: `${top}px`,
                    height: `${height}px`,
                    left: `calc(${col * pct}% + 2px)`,
                    width: `calc(${pct}% - 4px)`,
                  }}
                >
                  <div className="font-semibold text-sm leading-tight truncate">{event.title}</div>
                  <div className="text-xs opacity-90 mt-0.5 truncate">
                    {event.startTime} – {event.endTime}
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
