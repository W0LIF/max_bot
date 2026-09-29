/* ——— Единый источник типов. Соответствие контракту API (docs/API.md). ——— */

export type MoodValue = 'good' | 'ok' | 'bad'

/* Статусов ровно три — как в storage.TaskStatus на бэкенде.
   «Просрочено» — не статус, а вычисляемый признак (см. isOverdue). */
export type TaskStatus = 'new' | 'in_progress' | 'done'

/* Четыре квадранта Эйзенхауэра кодируются серверными флагами urgent/important. */
export type Priority = 'urgent_important' | 'important' | 'urgent' | 'later'

export type Task = {
  /* id приходят с сервера числами, в моках — строки. Сравнение только через sameId(). */
  id: string | number
  title: string
  subject: string
  deadline: string
  status: TaskStatus
  priority: Priority
  owner?: string
  group?: boolean
}

export type MoodEntry = {
  id: string | number
  day: string
  date: string
  value: number
  mood: MoodValue
  note?: string
}

export type Member = {
  id: string | number
  name: string
  group: string
  tasks: number
}

export type Note = {
  id: string | number
  text: string
  allDay: boolean
  start: string
  end: string
}

export type Settings = {
  notifications: boolean
  darkTheme: boolean
  language: 'ru' | 'en'
}

export type UserProfile = {
  name: string
  course: string
  group: string
}

/* ——— Хелперы ——— */

/** id из API — число, в моках — строка: сравниваем через строку. */
export function sameId(a: string | number, b: string | number): boolean {
  return String(a) === String(b)
}

/** Приоритет из серверных флагов urgent / important. */
export function priorityFromFlags(urgent: boolean, important: boolean): Priority {
  if (urgent && important) return 'urgent_important'
  if (important) return 'important'
  if (urgent) return 'urgent'
  return 'later'
}

const MONTHS = [
  'янв', 'фев', 'мар', 'апр', 'ма', 'июн', 'июл', 'авг', 'сен', 'окт', 'ноя', 'дек',
]

function startOfDay(date: Date): Date {
  return new Date(date.getFullYear(), date.getMonth(), date.getDate())
}

/**
 * Срок приходит либо человекочитаемой строкой («Сегодня · 18:30», «19 сентября · 09:00»),
 * либо ISO-датой с сервера. Разбираем оба варианта; если дату достать нельзя — null.
 */
export function parseDeadline(raw: string): Date | null {
  if (!raw) return null
  const value = raw.toLowerCase()

  const iso = value.match(/^(\d{4})-(\d{2})-(\d{2})/)
  if (iso) return new Date(Number(iso[1]), Number(iso[2]) - 1, Number(iso[3]))

  if (value.includes('сегодня')) return startOfDay(new Date())
  if (value.includes('вчера')) {
    const day = startOfDay(new Date())
    day.setDate(day.getDate() - 1)
    return day
  }
  if (value.includes('завтра')) {
    const day = startOfDay(new Date())
    day.setDate(day.getDate() + 1)
    return day
  }

  const ru = value.match(/(\d{1,2})\s*([а-яё]+)/)
  if (ru) {
    const month = MONTHS.findIndex((name) => ru[2].startsWith(name))
    if (month >= 0) return new Date(new Date().getFullYear(), month, Number(ru[1]))
  }

  const parsed = new Date(value)
  return Number.isNaN(parsed.getTime()) ? null : startOfDay(parsed)
}

/** Просрочка: срок раньше сегодняшнего дня и задача не выполнена. */
export function isOverdue(task: Task): boolean {
  if (task.status === 'done') return false
  const deadline = parseDeadline(task.deadline)
  if (!deadline) return false
  return deadline < startOfDay(new Date())
}
