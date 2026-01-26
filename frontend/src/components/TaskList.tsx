import { Calendar, Flag } from 'lucide-react';
import { Task } from '../screens/TasksScreen';

interface TaskListProps {
  tasks: Task[];
  onTaskSelect: (task: Task) => void;
  selectedTaskId?: string;
}

const TaskList = ({ tasks, onTaskSelect, selectedTaskId }: TaskListProps) => {
  const getStatusColor = (status: string) => {
    switch (status) {
      case 'in-progress':
        return 'bg-[#5C6BFF]';
      case 'completed':
        return 'bg-[#4CB782]';
      case 'overdue':
        return 'bg-[#FF8A65]';
      case 'planned':
        return 'bg-gray-400';
      default:
        return 'bg-gray-300';
    }
  };

  const getPriorityColor = (priority: string) => {
    switch (priority) {
      case 'high':
        return 'text-[#FF8A65]';
      case 'medium':
        return 'text-[#5C6BFF]';
      case 'low':
        return 'text-gray-400';
      default:
        return 'text-gray-400';
    }
  };

  const getStatusLabel = (status: string) => {
    switch (status) {
      case 'in-progress':
        return 'В работе';
      case 'completed':
        return 'Завершено';
      case 'overdue':
        return 'Просрочено';
      case 'planned':
        return 'Запланировано';
      default:
        return status;
    }
  };

  const formatDate = (dateString: string) => {
    const date = new Date(dateString);
    const today = new Date();
    const tomorrow = new Date(today);
    tomorrow.setDate(tomorrow.getDate() + 1);

    if (date.toDateString() === today.toDateString()) {
      return 'Сегодня';
    } else if (date.toDateString() === tomorrow.toDateString()) {
      return 'Завтра';
    } else {
      return date.toLocaleDateString('ru-RU', { day: 'numeric', month: 'short' });
    }
  };

  return (
    <div className="space-y-3">
      {tasks.map((task) => (
        <div
          key={task.id}
          onClick={() => onTaskSelect(task)}
          className={`bg-white p-5 rounded-2xl border-2 transition-all cursor-pointer hover:shadow-lg ${
            selectedTaskId === task.id
              ? 'border-[#5C6BFF] shadow-lg'
              : 'border-gray-100 hover:border-gray-200'
          }`}
        >
          <div className="flex items-start justify-between mb-3">
            <div className="flex-1">
              <h4 className="font-semibold text-[#2D2F31] mb-1">{task.title}</h4>
              <p className="text-sm text-gray-500 line-clamp-1">{task.description}</p>
            </div>
            <Flag size={16} className={getPriorityColor(task.priority)} fill="currentColor" />
          </div>

          <div className="flex items-center gap-2 mb-3">
            <span className={`px-2.5 py-1 rounded-lg text-xs font-medium text-white ${getStatusColor(task.status)}`}>
              {getStatusLabel(task.status)}
            </span>
            <div className="flex items-center gap-1 text-xs text-gray-500">
              <Calendar size={14} />
              <span>{formatDate(task.deadline)}</span>
            </div>
          </div>

          <div className="mb-3">
            <div className="flex items-center justify-between text-xs text-gray-500 mb-1">
              <span>Прогресс</span>
              <span className="font-medium">{task.progress}%</span>
            </div>
            <div className="w-full h-2 bg-gray-100 rounded-full overflow-hidden">
              <div
                className={`h-full rounded-full transition-all ${getStatusColor(task.status)}`}
                style={{ width: `${task.progress}%` }}
              />
            </div>
          </div>

          <div className="flex flex-wrap gap-1.5">
            {task.tags.map((tag, index) => (
              <span
                key={index}
                className="px-2 py-1 bg-[#F7F8FA] text-xs text-gray-600 rounded-md"
              >
                {tag}
              </span>
            ))}
          </div>
        </div>
      ))}
    </div>
  );
};

export default TaskList;
