import { useState } from 'react';
import { Search, Plus } from 'lucide-react';
import TaskList from '../components/TaskList';
import TaskDetail from '../components/TaskDetail';
import TaskFilters from '../components/TaskFilters';

export type TaskStatus = 'planned' | 'in-progress' | 'completed' | 'overdue';
export type TaskPriority = 'high' | 'medium' | 'low';
export type TaskCategory = 'work' | 'study' | 'personal' | 'finance' | 'health';

export interface Task {
  id: string;
  title: string;
  description: string;
  status: TaskStatus;
  priority: TaskPriority;
  category: TaskCategory;
  deadline: string;
  progress: number;
  tags: string[];
  notes?: string;
}

const TasksScreen = () => {
  const [selectedTask, setSelectedTask] = useState<Task | null>(null);
  const [filterCategory, setFilterCategory] = useState<string>('all');
  const [searchQuery, setSearchQuery] = useState('');

  const mockTasks: Task[] = [
    {
      id: '1',
      title: 'Подготовить презентацию для клиента',
      description: 'Создать презентацию по новому продукту',
      status: 'in-progress',
      priority: 'high',
      category: 'work',
      deadline: '2025-11-18',
      progress: 65,
      tags: ['срочно', 'клиент'],
    },
    {
      id: '2',
      title: 'Изучить новый фреймворк React',
      description: 'Пройти курс по React и TypeScript',
      status: 'planned',
      priority: 'medium',
      category: 'study',
      deadline: '2025-11-25',
      progress: 30,
      tags: ['обучение', 'разработка'],
    },
    {
      id: '3',
      title: 'Оплатить коммунальные услуги',
      description: 'Оплатить счета за ноябрь',
      status: 'planned',
      priority: 'high',
      category: 'finance',
      deadline: '2025-11-20',
      progress: 0,
      tags: ['платежи'],
    },
    {
      id: '4',
      title: 'Тренировка в спортзале',
      description: 'Кардио и силовая тренировка',
      status: 'completed',
      priority: 'medium',
      category: 'health',
      deadline: '2025-11-16',
      progress: 100,
      tags: ['спорт', 'здоровье'],
    },
    {
      id: '5',
      title: 'Написать статью для блога',
      description: 'Статья о продуктивности',
      status: 'overdue',
      priority: 'low',
      category: 'personal',
      deadline: '2025-11-15',
      progress: 45,
      tags: ['блог', 'контент'],
    },
  ];

  const filteredTasks = mockTasks.filter((task) => {
    const matchesCategory = filterCategory === 'all' || task.category === filterCategory || task.status === filterCategory;
    const matchesSearch = task.title.toLowerCase().includes(searchQuery.toLowerCase());
    return matchesCategory && matchesSearch;
  });

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
              <p className="text-xs md:text-sm text-gray-500 mt-1">{filteredTasks.length} задач</p>
            </div>
            <button className="flex items-center justify-center gap-2 px-4 py-2.5 bg-[#5C6BFF] text-white rounded-xl hover:bg-[#4C5BEF] transition-colors shadow-lg shadow-[#5C6BFF]/20 text-sm md:text-base">
              <Plus size={18} />
              <span className="font-medium">Новая</span>
            </button>
          </div>
          <TaskList tasks={filteredTasks} onTaskSelect={setSelectedTask} selectedTaskId={selectedTask?.id} />
        </div>
      </div>

      {selectedTask && (
        <>
          <div className="md:hidden fixed inset-0 bg-black/50 z-40" onClick={() => setSelectedTask(null)} />
          <div className="md:w-96 fixed md:relative bottom-0 left-0 right-0 md:bottom-auto bg-white rounded-t-2xl md:rounded-none md:border-l border-gray-200 overflow-auto max-h-[80vh] md:max-h-full z-40">
            <TaskDetail task={selectedTask} onClose={() => setSelectedTask(null)} />
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
