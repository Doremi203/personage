import { X, Calendar, Flag, FolderOpen, Clock, Tag, FileText, CheckCircle, Pause, Trash2, Loader2 } from 'lucide-react';
import { Task } from '../screens/TasksScreen';

interface TaskDetailProps {
  task: Task;
  onClose: () => void;
  onComplete?: () => Promise<void>;
  onPostpone?: () => Promise<void>;
  onDelete?: () => Promise<void>;
  actionLoading?: boolean;
}

const TaskDetail = ({ task, onClose, onComplete, onPostpone, onDelete, actionLoading }: TaskDetailProps) => {
  const getPriorityLabel = (priority: string) => {
    switch (priority) {
      case 'high':
        return 'Высокий';
      case 'medium':
        return 'Средний';
      case 'low':
        return 'Низкий';
      default:
        return priority;
    }
  };

  const getCategoryLabel = (category: string) => {
    switch (category) {
      case 'work':
        return 'Работа';
      case 'study':
        return 'Учёба';
      case 'personal':
        return 'Личное';
      case 'finance':
        return 'Финансы';
      case 'health':
        return 'Здоровье';
      default:
        return category;
    }
  };

  const formatDate = (dateString: string) => {
    const date = new Date(dateString);
    return date.toLocaleDateString('ru-RU', {
      day: 'numeric',
      month: 'long',
      year: 'numeric'
    });
  };

  return (
    <div className="h-full flex flex-col">
      <div className="p-6 border-b border-gray-200 flex items-center justify-between">
        <h3 className="font-semibold text-lg text-[#2D2F31]">Детали задачи</h3>
        <button
          onClick={onClose}
          className="p-2 hover:bg-gray-100 rounded-lg transition-colors"
        >
          <X size={20} className="text-gray-500" />
        </button>
      </div>

      <div className="flex-1 overflow-auto p-6">
        <div className="mb-6">
          <h2 className="text-xl font-bold text-[#2D2F31] mb-2">{task.title}</h2>
          <p className="text-gray-600">{task.description}</p>
        </div>

        <div className="space-y-4 mb-6">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 bg-[#5C6BFF]/10 rounded-lg flex items-center justify-center">
              <Flag size={18} className="text-[#5C6BFF]" />
            </div>
            <div>
              <p className="text-xs text-gray-500">Приоритет</p>
              <p className="text-sm font-medium text-[#2D2F31]">{getPriorityLabel(task.priority)}</p>
            </div>
          </div>

          <div className="flex items-center gap-3">
            <div className="w-10 h-10 bg-[#4CB782]/10 rounded-lg flex items-center justify-center">
              <FolderOpen size={18} className="text-[#4CB782]" />
            </div>
            <div>
              <p className="text-xs text-gray-500">Категория</p>
              <p className="text-sm font-medium text-[#2D2F31]">{getCategoryLabel(task.category)}</p>
            </div>
          </div>

          <div className="flex items-center gap-3">
            <div className="w-10 h-10 bg-[#FF8A65]/10 rounded-lg flex items-center justify-center">
              <Calendar size={18} className="text-[#FF8A65]" />
            </div>
            <div>
              <p className="text-xs text-gray-500">Дедлайн</p>
              <p className="text-sm font-medium text-[#2D2F31]">{formatDate(task.deadline)}</p>
            </div>
          </div>

          <div className="flex items-center gap-3">
            <div className="w-10 h-10 bg-gray-100 rounded-lg flex items-center justify-center">
              <Clock size={18} className="text-gray-600" />
            </div>
            <div className="flex-1">
              <p className="text-xs text-gray-500 mb-1">Прогресс</p>
              <div className="flex items-center gap-3">
                <div className="flex-1 h-2 bg-gray-100 rounded-full overflow-hidden">
                  <div
                    className="h-full bg-[#5C6BFF] rounded-full transition-all"
                    style={{ width: `${task.progress}%` }}
                  />
                </div>
                <span className="text-sm font-medium text-[#2D2F31]">{task.progress}%</span>
              </div>
            </div>
          </div>
        </div>

        <div className="mb-6">
          <div className="flex items-center gap-2 mb-3">
            <Tag size={16} className="text-gray-500" />
            <p className="text-sm font-medium text-gray-700">Теги</p>
          </div>
          <div className="flex flex-wrap gap-2">
            {task.tags.map((tag, index) => (
              <span
                key={index}
                className="px-3 py-1.5 bg-[#F7F8FA] text-sm text-gray-700 rounded-lg border border-gray-200"
              >
                {tag}
              </span>
            ))}
          </div>
        </div>

        <div className="mb-6">
          <div className="flex items-center gap-2 mb-3">
            <FileText size={16} className="text-gray-500" />
            <p className="text-sm font-medium text-gray-700">Заметки</p>
          </div>
          <textarea
            placeholder="Добавьте заметки к задаче..."
            className="w-full h-32 p-3 bg-[#F7F8FA] border border-gray-200 rounded-xl resize-none focus:outline-none focus:ring-2 focus:ring-[#5C6BFF] focus:border-transparent"
            defaultValue={task.notes}
          />
        </div>

        <div>
          <p className="text-sm font-medium text-gray-700 mb-3">Связанные события</p>
          <div className="p-4 bg-[#F7F8FA] rounded-xl border border-gray-200 text-center">
            <p className="text-sm text-gray-500">Нет связанных событий</p>
          </div>
        </div>
      </div>

      <div className="p-6 border-t border-gray-200 space-y-2">
        <div className="grid grid-cols-2 gap-2">
          <button
            onClick={() => onComplete?.()}
            disabled={actionLoading}
            className="flex items-center justify-center gap-2 px-4 py-2.5 bg-[#4CB782] text-white rounded-xl hover:bg-[#3CA672] transition-colors text-sm font-medium disabled:opacity-50"
          >
            {actionLoading ? <Loader2 size={16} className="animate-spin" /> : <CheckCircle size={16} />}
            <span>Завершить</span>
          </button>
          <button
            onClick={() => onPostpone?.()}
            disabled={actionLoading}
            className="flex items-center justify-center gap-2 px-4 py-2.5 bg-gray-100 text-gray-700 rounded-xl hover:bg-gray-200 transition-colors text-sm font-medium disabled:opacity-50"
          >
            {actionLoading ? <Loader2 size={16} className="animate-spin" /> : <Pause size={16} />}
            <span>Отложить</span>
          </button>
        </div>
        <button
          onClick={() => onDelete?.()}
          disabled={actionLoading}
          className="w-full flex items-center justify-center gap-2 px-4 py-2.5 border-2 border-red-200 text-red-600 rounded-xl hover:bg-red-50 transition-colors text-sm font-medium disabled:opacity-50"
        >
          {actionLoading ? <Loader2 size={16} className="animate-spin" /> : <Trash2 size={16} />}
          <span>Удалить</span>
        </button>
      </div>
    </div>
  );
};

export default TaskDetail;
