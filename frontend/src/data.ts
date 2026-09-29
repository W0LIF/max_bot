import type { MoodValue } from './ui/kit'

/* ——— Модель данных ——— */
export type TaskStatus = 'new' | 'in_progress' | 'done' | 'overdue' | 'frozen'
export type Priority = 'urgent' | 'important' | 'later'

export type Task = {
  id: string
  title: string
  subject: string
  deadline: string
  status: TaskStatus
  priority: Priority
  owner?: string
  group?: boolean
}

export type MoodEntry = {
  id: string
  day: string
  date: string
  value: number
  mood: MoodValue
  note?: string
}

export type Member = {
  id: string
  name: string
  group: string
  tasks: number
}

export type Achievement = {
  id: string
  title: string
  icon: 'sparkle' | 'moon' | 'target' | 'star' | 'telescope'
  locked: boolean
  x: number
  y: number
}

export type Note = {
  id: string
  text: string
  allDay: boolean
  start: string
  end: string
}

export type Faq = { q: string; a: string }

/* ——— Справочники ——— */
export const STATUS_META: Record<TaskStatus, { label: string; color: string }> = {
  new: { label: 'Новая', color: 'var(--blue-light)' },
  in_progress: { label: 'В процессе', color: 'var(--st-progress)' },
  done: { label: 'Выполнено', color: 'var(--st-done)' },
  overdue: { label: 'Просрочено', color: 'var(--st-overdue)' },
  frozen: { label: 'На паузе', color: 'var(--st-frozen)' },
}

export const PRIORITY_META: Record<Priority, { label: string; tone: 'urgent' | 'important' | 'later' }> = {
  urgent: { label: 'Срочно', tone: 'urgent' },
  important: { label: 'Важно', tone: 'important' },
  later: { label: 'Позже', tone: 'later' },
}

export const MOOD_META: Record<MoodValue, { label: string; color: string }> = {
  good: { label: 'Хорошее', color: 'var(--st-done)' },
  ok: { label: 'Нормальное', color: 'var(--st-progress)' },
  bad: { label: 'Плохое', color: 'var(--st-overdue)' },
}

export const SUBJECTS = [
  'Языки программирования',
  'Физика',
  'ОТЦ',
  'Математика',
  'История',
  'Английский язык',
]

/* ——— Мок-данные (как в макете) ——— */
export const USER = { name: 'Аня', course: '1 курс', group: 'ИКТн-54' }

export const INITIAL_TASKS: Task[] = [
  {
    id: 't1',
    title: 'Лабораторная №3',
    subject: 'Языки программирования',
    deadline: 'Сегодня · 18:30',
    status: 'in_progress',
    priority: 'urgent',
  },
  {
    id: 't2',
    title: 'Курсовая: глава 2',
    subject: 'Физика',
    deadline: 'Завтра · 12:00',
    status: 'new',
    priority: 'important',
  },
  {
    id: 't3',
    title: 'Курсовая: расчёт',
    subject: 'ОТЦ',
    deadline: '19 сентября · 09:00',
    status: 'overdue',
    priority: 'important',
  },
  {
    id: 't4',
    title: 'Домашнее задание №7',
    subject: 'Математика',
    deadline: '15 сентября',
    status: 'done',
    priority: 'later',
  },
  {
    id: 't5',
    title: 'Эссе по истории',
    subject: 'История',
    deadline: '22 сентября',
    status: 'frozen',
    priority: 'later',
  },
]

export const GROUP_TASKS: Task[] = [
  {
    id: 'g1',
    title: 'Лабораторная №3',
    subject: 'Языки программирования',
    deadline: 'Сегодня · 18:30',
    status: 'done',
    priority: 'urgent',
    owner: 'Аня А.',
    group: true,
  },
  {
    id: 'g2',
    title: 'Отчёт по практике',
    subject: 'ОТЦ',
    deadline: '20 сентября',
    status: 'in_progress',
    priority: 'important',
    owner: 'Маша Г.',
    group: true,
  },
  {
    id: 'g3',
    title: 'Курсовая: глава 2',
    subject: 'Физика',
    deadline: 'Завтра · 12:00',
    status: 'overdue',
    priority: 'urgent',
    owner: 'Олег В.',
    group: true,
  },
]

export const MEMBERS: Member[] = [
  { id: 'm1', name: 'Аня А.', group: 'ИКТн-54', tasks: 6 },
  { id: 'm2', name: 'Настя В.', group: 'ИКТн-54', tasks: 6 },
  { id: 'm3', name: 'Олег В.', group: 'ИКТн-54', tasks: 6 },
  { id: 'm4', name: 'Маша Г.', group: 'ИКТн-54', tasks: 7 },
]

export const MOOD_HISTORY: MoodEntry[] = [
  { id: 'd1', day: 'Пн', date: '14.09', value: 16, mood: 'good' },
  { id: 'd2', day: 'Вт', date: '15.09', value: 13, mood: 'ok' },
  { id: 'd3', day: 'Ср', date: '16.09', value: 18, mood: 'good' },
  { id: 'd4', day: 'Чт', date: '17.09', value: 12, mood: 'bad' },
  { id: 'd5', day: 'Пт', date: '18.09', value: 15, mood: 'ok' },
  { id: 'd6', day: 'Сб', date: '19.09', value: 19, mood: 'good' },
  { id: 'd7', day: 'Вс', date: '20.09', value: 14, mood: 'ok' },
]

export const ACHIEVEMENTS: Achievement[] = [
  { id: 'a1', title: 'Первая звезда', icon: 'sparkle', locked: false, x: 26, y: 12 },
  { id: 'a2', title: 'Загадка', icon: 'moon', locked: false, x: 68, y: 27 },
  { id: 'a3', title: 'Исследователь', icon: 'target', locked: true, x: 30, y: 45 },
  { id: 'a4', title: 'Мастер', icon: 'star', locked: true, x: 66, y: 64 },
  { id: 'a5', title: 'Телескоп', icon: 'telescope', locked: true, x: 32, y: 85 },
]

export const FAQ: Faq[] = [
  {
    q: 'Как добавить задачу?',
    a: 'На главной нажмите на синюю кнопку «+». Задачу можно ввести вручную или распознать с фотографии расписания.',
  },
  {
    q: 'Как распознаётся скан?',
    a: 'Приложение выделяет текст на фото, определяет предмет, название работы и срок — остаётся только проверить и сохранить.',
  },
  {
    q: 'Как распределяется нагрузка?',
    a: '«Твой баланс» считает задачи на неделю, дедлайны и отметки настроения. Если нагрузка выше 2/3, карточка подсветится розовым.',
  },
  {
    q: 'Как поставить дедлайн?',
    a: 'В форме добавления задачи откройте поле «Срок» — там же можно отметить, что задача занимает весь день.',
  },
  {
    q: 'Где найти общую задачу?',
    a: 'Нижняя панель → «Группа». Выберите однокурсников и создайте группу: задания появятся в общем списке.',
  },
  {
    q: 'Что такое «баланс»?',
    a: 'Это три показателя: нагрузка, настроение и прогресс по задачам. Они помогают вовремя снизить темп.',
  },
]

export const REMINDERS = [
  { id: 'r1', title: 'Купить учебник по ОТЦ до 20.09', time: '12:30' },
  { id: 'r2', title: 'Консультация по математике', time: '16:00' },
]
