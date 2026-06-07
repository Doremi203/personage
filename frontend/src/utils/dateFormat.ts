export const RU_MONTHS_GEN = [
  'января', 'февраля', 'марта', 'апреля', 'мая', 'июня',
  'июля', 'августа', 'сентября', 'октября', 'ноября', 'декабря',
];

export const RU_WEEKDAYS_SHORT = ['Пн', 'Вт', 'Ср', 'Чт', 'Пт', 'Сб', 'Вс'];

export function startOfDay(d: Date): Date {
  const x = new Date(d);
  x.setHours(0, 0, 0, 0);
  return x;
}

export function sameDay(a: Date, b: Date): boolean {
  return startOfDay(a).getTime() === startOfDay(b).getTime();
}

export function toApiDateParam(d: Date): string {
  const dd = String(d.getDate()).padStart(2, '0');
  const mm = String(d.getMonth() + 1).padStart(2, '0');
  return `${dd}-${mm}-${d.getFullYear()}`;
}

export function toHHMM(d: Date): string {
  return `${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}`;
}

// "Сегодня" / "Завтра" / "Вчера" / "20 мая" / "20 мая 2025" — день без времени.
export function formatDayLabel(d: Date): string {
  const today = startOfDay(new Date());
  const tomorrow = new Date(today);
  tomorrow.setDate(today.getDate() + 1);
  const yesterday = new Date(today);
  yesterday.setDate(today.getDate() - 1);
  const day = startOfDay(d);
  if (day.getTime() === today.getTime())     return 'Сегодня';
  if (day.getTime() === tomorrow.getTime())  return 'Завтра';
  if (day.getTime() === yesterday.getTime()) return 'Вчера';
  const sameYear = d.getFullYear() === today.getFullYear();
  return `${d.getDate()} ${RU_MONTHS_GEN[d.getMonth()]}${sameYear ? '' : ' ' + d.getFullYear()}`;
}

// "Сегодня, 14:00" / "20 мая, 14:00" — день со временем (для слотов расписания).
export function formatDateTimeLabel(d: Date): string {
  return `${formatDayLabel(d)}, ${toHHMM(d)}`;
}

// «Безвременной» дедлайн (только дата): LLM кодирует его как конец дня по Москве —
// 23:59:59, а заданный вручную через админку может быть полночью. В обоих случаях
// время синтетическое, поэтому показываем только день. Часы читаем в локали браузера
// (как и остальной UI — приложение ориентировано на Москву).
export function formatDeadlineLabel(d: Date): string {
  const h = d.getHours();
  const m = d.getMinutes();
  const allDay = (h === 23 && m === 59) || (h === 0 && m === 0);
  return allDay ? formatDayLabel(d) : formatDateTimeLabel(d);
}
