import { TrendingUp, CheckCircle, AlertTriangle, Clock, Target, Activity } from 'lucide-react';

const WeeklyDigest = () => {
  return (
    <div className="p-6">
      <div className="mb-6">
        <h3 className="text-xl font-bold text-[#2D2F31] mb-2">Дайджест недели</h3>
        <p className="text-sm text-gray-500">14 ноября - 20 ноября</p>
      </div>

      <div className="grid grid-cols-2 gap-3 mb-6">
        <div className="bg-gradient-to-br from-[#4CB782] to-[#3CA672] rounded-xl p-4 text-white">
          <div className="flex items-center gap-2 mb-2">
            <CheckCircle size={18} />
            <span className="text-xs font-medium opacity-90">Выполнено</span>
          </div>
          <p className="text-3xl font-bold mb-1">24</p>
          <p className="text-xs opacity-80">задачи</p>
        </div>

        <div className="bg-gradient-to-br from-[#5C6BFF] to-[#4C5BEF] rounded-xl p-4 text-white">
          <div className="flex items-center gap-2 mb-2">
            <Clock size={18} />
            <span className="text-xs font-medium opacity-90">Время</span>
          </div>
          <p className="text-3xl font-bold mb-1">42</p>
          <p className="text-xs opacity-80">часа фокуса</p>
        </div>

        <div className="bg-gradient-to-br from-[#FF8A65] to-[#FF7A55] rounded-xl p-4 text-white">
          <div className="flex items-center gap-2 mb-2">
            <AlertTriangle size={18} />
            <span className="text-xs font-medium opacity-90">Пропущено</span>
          </div>
          <p className="text-3xl font-bold mb-1">2</p>
          <p className="text-xs opacity-80">задачи</p>
        </div>

        <div className="bg-gradient-to-br from-gray-700 to-[#2D2F31] rounded-xl p-4 text-white">
          <div className="flex items-center gap-2 mb-2">
            <Target size={18} />
            <span className="text-xs font-medium opacity-90">Цель</span>
          </div>
          <p className="text-3xl font-bold mb-1">85%</p>
          <p className="text-xs opacity-80">достигнута</p>
        </div>
      </div>

      <div className="mb-6">
        <div className="flex items-center gap-2 mb-4">
          <Activity size={18} className="text-[#5C6BFF]" />
          <h4 className="font-semibold text-[#2D2F31]">Активность по дням</h4>
        </div>
        <div className="space-y-3">
          {[
            { day: 'Пн', tasks: 6, hours: 8, progress: 85 },
            { day: 'Вт', tasks: 5, hours: 7, progress: 90 },
            { day: 'Ср', tasks: 4, hours: 6, progress: 75 },
            { day: 'Чт', tasks: 5, hours: 8, progress: 80 },
            { day: 'Пт', tasks: 4, hours: 5, progress: 70 },
            { day: 'Сб', tasks: 0, hours: 0, progress: 0 },
            { day: 'Вс', tasks: 0, hours: 8, progress: 100 },
          ].map((item, index) => (
            <div key={index} className="bg-[#F7F8FA] rounded-xl p-3 border border-gray-200">
              <div className="flex items-center justify-between mb-2">
                <span className="font-semibold text-sm text-[#2D2F31]">{item.day}</span>
                <div className="flex items-center gap-3 text-xs text-gray-600">
                  <span>{item.tasks} задач</span>
                  <span>•</span>
                  <span>{item.hours}ч</span>
                </div>
              </div>
              <div className="w-full h-2 bg-white rounded-full overflow-hidden">
                <div
                  className="h-full bg-gradient-to-r from-[#5C6BFF] to-[#7C8CFF] rounded-full transition-all"
                  style={{ width: `${item.progress}%` }}
                />
              </div>
            </div>
          ))}
        </div>
      </div>

      <div className="space-y-3">
        <div className="flex items-center gap-2 mb-3">
          <TrendingUp size={18} className="text-[#4CB782]" />
          <h4 className="font-semibold text-[#2D2F31]">Важные достижения</h4>
        </div>

        <div className="p-4 bg-gradient-to-br from-[#4CB782]/10 to-[#4CB782]/5 rounded-xl border border-[#4CB782]/20">
          <div className="flex items-center gap-2 mb-2">
            <CheckCircle size={16} className="text-[#4CB782]" />
            <span className="font-medium text-sm text-[#2D2F31]">Отличная продуктивность!</span>
          </div>
          <p className="text-xs text-gray-600">
            Вы выполнили на 15% больше задач, чем на прошлой неделе
          </p>
        </div>

        <div className="p-4 bg-gradient-to-br from-[#FF8A65]/10 to-[#FF8A65]/5 rounded-xl border border-[#FF8A65]/20">
          <div className="flex items-center gap-2 mb-2">
            <AlertTriangle size={16} className="text-[#FF8A65]" />
            <span className="font-medium text-sm text-[#2D2F31]">Пропущена тренировка</span>
          </div>
          <p className="text-xs text-gray-600">
            Вы пропустили тренировку в среду. Не забудьте о здоровье!
          </p>
        </div>

        <div className="p-4 bg-gradient-to-br from-[#5C6BFF]/10 to-[#5C6BFF]/5 rounded-xl border border-[#5C6BFF]/20">
          <div className="flex items-center gap-2 mb-2">
            <Clock size={16} className="text-[#5C6BFF]" />
            <span className="font-medium text-sm text-[#2D2F31]">Скоро дедлайн</span>
          </div>
          <p className="text-xs text-gray-600">
            Осталось 2 дня до дедлайна по задаче "Подготовить презентацию"
          </p>
        </div>
      </div>
    </div>
  );
};

export default WeeklyDigest;
