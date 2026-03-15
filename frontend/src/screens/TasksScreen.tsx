import { useState, useEffect, useCallback } from 'react';
import { Search, Plus, Loader2 } from 'lucide-react';
import TaskList from '../components/TaskList';
import TaskDetail from '../components/TaskDetail';
import TaskFilters from '../components/TaskFilters';
import {
  listTasks,
  completeTask,
  postponeTask,
  deleteTask,
  ApiTaskItem,
  ApiTaskStatus,
  ApiTaskPriority,
  ApiTaskCategory,
  ApiTaskStatusFilter,
  ApiTaskCategoryFilter,
} from '../utils/taskerService';

export type TaskStatus = 'unplanned' | 'planned' | 'completed';
export type TaskPriority = 'high' | 'medium' | 'low';
export type TaskCategory = 'work' | 'study' | 'personal';

export interface Task {
  id: string;
  title: string;
  description: string;
  status: TaskStatus;
  priority: TaskPriority;
  category: TaskCategory;
  deadline: string;
  startTime: string;
  endTime: string;
  progress: number;
  tags: string[];
  notes?: string;
}

function mapStatus(status: string): TaskStatus {
  if (status === ApiTaskStatus.COMPLETED) return 'completed';
  if (status === ApiTaskStatus.UNPLANNED) return 'unplanned';
  return 'planned';
}

function mapPriority(priority: string): TaskPriority {
  if (priority === ApiTaskPriority.HIGH) return 'high';
  if (priority === ApiTaskPriority.LOW) return 'low';
  return 'medium';
}

function mapCategory(category: string): TaskCategory {
  if (category === ApiTaskCategory.WORK) return 'work';
  if (category === ApiTaskCategory.STUDY) return 'study';
  return 'personal';
}

function mapApiTask(apiTask: ApiTaskItem): Task {
  const toDateString = (ts?: string) =>
    ts ? new Date(ts).toISOString().split('T')[0] : '';
  return {
    id: apiTask.id,
    title: apiTask.title,
    description: apiTask.description,
    status: mapStatus(apiTask.status),
    priority: mapPriority(apiTask.priority),
    category: mapCategory(apiTask.category),
    deadline: toDateString(apiTask.deadline),
    startTime: toDateString(apiTask.startTime),
    endTime: toDateString(apiTask.endTime),
    progress: apiTask.status === ApiTaskStatus.COMPLETED ? 100 : 0,
    tags: [],
  };
}

function getApiFilters(filter: string): {
  status?: string;
  category?: string;
} {
  switch (filter) {
    case 'unplanned':
      return { status: ApiTaskStatusFilter.UNPLANNED };
    case 'planned':
      return { status: ApiTaskStatusFilter.PLANNED };
    case 'completed':
      return { status: ApiTaskStatusFilter.COMPLETED };
    case 'work':
      return { category: ApiTaskCategoryFilter.WORK };
    case 'study':
      return { category: ApiTaskCategoryFilter.STUDY };
    case 'personal':
      return { category: ApiTaskCategoryFilter.PERSONAL };
    default:
      return {};
  }
}

const TasksScreen = () => {
  const [selectedTask, setSelectedTask] = useState<Task | null>(null);
  const [filterCategory, setFilterCategory] = useState<string>('all');
  const [searchQuery, setSearchQuery] = useState('');
  const [debouncedSearch, setDebouncedSearch] = useState('');
  const [tasks, setTasks] = useState<Task[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [actionLoading, setActionLoading] = useState(false);

  useEffect(() => {
    const timer = setTimeout(() => setDebouncedSearch(searchQuery), 300);
    return () => clearTimeout(timer);
  }, [searchQuery]);

  const fetchTasks = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const filters = getApiFilters(filterCategory);
      const response = await listTasks({
        ...filters,
        text: debouncedSearch || undefined,
        pageSize: 20,
        page: 1,
      });
      const mapped = (response.tasks ?? []).map(mapApiTask);
      setTasks(mapped);
    } catch (err) {
      setError(
        err instanceof Error ? err.message : 'Не удалось загрузить задачи',
      );
    } finally {
      setLoading(false);
    }
  }, [filterCategory, debouncedSearch]);

  useEffect(() => {
    void fetchTasks();
  }, [fetchTasks]);

  const handleComplete = useCallback(async () => {
    if (!selectedTask) return;
    setActionLoading(true);
    try {
      await completeTask(selectedTask.id);
      await fetchTasks();
      setSelectedTask(null);
    } catch (err) {
      setError(
        err instanceof Error ? err.message : 'Не удалось завершить задачу',
      );
    } finally {
      setActionLoading(false);
    }
  }, [selectedTask, fetchTasks]);

  const handlePostpone = useCallback(async () => {
    if (!selectedTask) return;
    setActionLoading(true);
    try {
      await postponeTask(selectedTask.id);
      await fetchTasks();
      setSelectedTask(null);
    } catch (err) {
      setError(
        err instanceof Error ? err.message : 'Не удалось отложить задачу',
      );
    } finally {
      setActionLoading(false);
    }
  }, [selectedTask, fetchTasks]);

  const handleDelete = useCallback(async () => {
    if (!selectedTask) return;
    setActionLoading(true);
    try {
      await deleteTask(selectedTask.id);
      await fetchTasks();
      setSelectedTask(null);
    } catch (err) {
      setError(
        err instanceof Error ? err.message : 'Не удалось удалить задачу',
      );
    } finally {
      setActionLoading(false);
    }
  }, [selectedTask, fetchTasks]);

  return (
    <div className="h-full flex flex-col md:flex-row md:pt-0 pt-16">
      <div className="hidden md:flex md:w-80 bg-white border-r border-gray-200 flex-col">
        <div className="p-6 border-b border-gray-200">
          <h2 className="text-2xl font-bold text-[#2D2F31] mb-4">Задачи</h2>
          <div className="relative">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" size={18} />
            <input
              type="text"
              placeholder="Поиск задач..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="w-full pl-10 pr-4 py-2.5 bg-[#F7F8FA] border border-gray-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-[#5C6BFF] focus:border-transparent"
            />
          </div>
        </div>
        <TaskFilters currentFilter={filterCategory} onFilterChange={setFilterCategory} />
      </div>

      <div className="flex-1 overflow-auto">
        <div className="p-4 md:p-8">
          <div className="mb-4 md:mb-6">
            <h2 className="md:hidden text-xl font-bold text-[#2D2F31] mb-4">Задачи</h2>
            <div className="relative md:hidden mb-4">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" size={18} />
              <input
                type="text"
                placeholder="Поиск задач..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="w-full pl-10 pr-4 py-2.5 bg-[#F7F8FA] border border-gray-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-[#5C6BFF] focus:border-transparent text-sm"
              />
            </div>
          </div>
          <div className="flex flex-col md:flex-row md:items-center md:justify-between gap-4 mb-4 md:mb-6">
            <div>
              <h3 className="text-lg md:text-xl font-semibold text-[#2D2F31]">
                {filterCategory === 'all' ? 'Все задачи' : 'Фильтрованные'}
              </h3>
              <p className="text-xs md:text-sm text-gray-500 mt-1">{tasks.length} задач</p>
            </div>
            <button className="flex items-center justify-center gap-2 px-4 py-2.5 bg-[#5C6BFF] text-white rounded-xl hover:bg-[#4C5BEF] transition-colors shadow-lg shadow-[#5C6BFF]/20 text-sm md:text-base">
              <Plus size={18} />
              <span className="font-medium">Новая</span>
            </button>
          </div>
          {loading ? (
            <div className="flex items-center justify-center py-16 text-gray-400">
              <Loader2 size={32} className="animate-spin" />
            </div>
          ) : error ? (
            <div className="py-8 text-center">
              <p className="text-red-500 text-sm mb-3">{error}</p>
              <button
                onClick={() => void fetchTasks()}
                className="px-4 py-2 bg-[#5C6BFF] text-white rounded-xl text-sm hover:bg-[#4C5BEF] transition-colors"
              >
                Повторить
              </button>
            </div>
          ) : (
            <TaskList tasks={tasks} onTaskSelect={setSelectedTask} selectedTaskId={selectedTask?.id} />
          )}
        </div>
      </div>

      {selectedTask && (
        <>
          <div className="md:hidden fixed inset-0 bg-black/50 z-40" onClick={() => setSelectedTask(null)} />
          <div className="md:w-96 fixed md:relative bottom-0 left-0 right-0 md:bottom-auto bg-white rounded-t-2xl md:rounded-none md:border-l border-gray-200 overflow-auto max-h-[80vh] md:max-h-full z-40">
            <TaskDetail
              task={selectedTask}
              onClose={() => setSelectedTask(null)}
              onComplete={handleComplete}
              onPostpone={handlePostpone}
              onDelete={handleDelete}
              actionLoading={actionLoading}
            />
          </div>
        </>
      )}

      {!selectedTask && (
        <div className="md:flex hidden md:w-80 bg-white border-l border-gray-200 flex-col">
          <div className="p-4 text-center text-gray-500">
            <p>Выберите задачу для просмотра деталей</p>
          </div>
        </div>
      )}
    </div>
  );
};

export default TasksScreen;
