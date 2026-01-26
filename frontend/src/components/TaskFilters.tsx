import { ListTodo, Clock, PlayCircle, CheckCircle2, AlertCircle, Briefcase, GraduationCap, User, DollarSign, Heart } from 'lucide-react';

interface TaskFiltersProps {
  currentFilter: string;
  onFilterChange: (filter: string) => void;
}

const TaskFilters = ({ currentFilter, onFilterChange }: TaskFiltersProps) => {
  const statusFilters = [
    { id: 'all', label: 'Все задачи', icon: ListTodo },
    { id: 'planned', label: 'Запланированные', icon: Clock },
    { id: 'in-progress', label: 'В работе', icon: PlayCircle },
    { id: 'completed', label: 'Завершённые', icon: CheckCircle2 },
    { id: 'overdue', label: 'Просроченные', icon: AlertCircle },
  ];

  const categoryFilters = [
    { id: 'work', label: 'Работа', icon: Briefcase },
    { id: 'study', label: 'Учёба', icon: GraduationCap },
    { id: 'personal', label: 'Личное', icon: User },
    { id: 'finance', label: 'Финансы', icon: DollarSign },
    { id: 'health', label: 'Здоровье', icon: Heart },
  ];

  return (
    <div className="flex-1 overflow-auto p-4">
      <div className="mb-6">
        <h3 className="text-xs font-semibold text-gray-500 uppercase tracking-wider mb-3">Статус</h3>
        {statusFilters.map((filter) => {
          const Icon = filter.icon;
          const isActive = currentFilter === filter.id;
          return (
            <button
              key={filter.id}
              onClick={() => onFilterChange(filter.id)}
              className={`w-full flex items-center gap-3 px-3 py-2.5 rounded-lg mb-1 transition-all ${
                isActive
                  ? 'bg-[#5C6BFF]/10 text-[#5C6BFF]'
                  : 'text-gray-600 hover:bg-gray-50'
              }`}
            >
              <Icon size={18} />
              <span className="text-sm font-medium">{filter.label}</span>
            </button>
          );
        })}
      </div>

      <div>
        <h3 className="text-xs font-semibold text-gray-500 uppercase tracking-wider mb-3">Категории</h3>
        {categoryFilters.map((filter) => {
          const Icon = filter.icon;
          const isActive = currentFilter === filter.id;
          return (
            <button
              key={filter.id}
              onClick={() => onFilterChange(filter.id)}
              className={`w-full flex items-center gap-3 px-3 py-2.5 rounded-lg mb-1 transition-all ${
                isActive
                  ? 'bg-[#5C6BFF]/10 text-[#5C6BFF]'
                  : 'text-gray-600 hover:bg-gray-50'
              }`}
            >
              <Icon size={18} />
              <span className="text-sm font-medium">{filter.label}</span>
            </button>
          );
        })}
      </div>
    </div>
  );
};

export default TaskFilters;
