import type { Member, MoodEntry, MoodValue, Note, Task, TaskStatus } from './types'
import { priorityFromFlags } from './types'
import { getInitData } from './webapp'

const API_BASE = (import.meta.env.VITE_API_BASE || import.meta.env.VITE_API_URL || '/api').replace(/\/$/, '')

export class ApiError extends Error {
  readonly status: number

  constructor(message: string, status: number) {
    super(message)
    this.status = status
    this.name = 'ApiError'
  }
}

type ApiTask = {
  id: number
  user_id: number
  title: string
  subject: string
  deadline: string
  urgent: boolean
  important: boolean
  status: TaskStatus
  group?: boolean
}

type ApiMood = {
  id: number
  value: MoodValue
  note?: string
  created_at: string
}

type ApiNote = {
  id: number
  text: string
  allDay: boolean
  start: string
  end: string
}

export type ApiProfile = {
  id: number
  name: string
  consent: boolean
  onboarded: boolean
  reminders_on: boolean
}

async function request<T>(path: string, init: RequestInit = {}, authenticated = true): Promise<T> {
  const headers = new Headers(init.headers)
  if (!(init.body instanceof FormData) && init.body !== undefined) {
    headers.set('Content-Type', 'application/json')
  }

  if (authenticated) {
    const initData = getInitData()
    if (!initData) throw new ApiError('Откройте Телескоп внутри MAX, чтобы войти.', 401)
    headers.set('X-Max-Init-Data', initData)
  }

  let response: Response
  try {
    response = await fetch(`${API_BASE}${path}`, { ...init, headers })
  } catch {
    throw new ApiError('Не удалось подключиться. Проверьте интернет и повторите попытку.', 0)
  }

  if (!response.ok) {
    let message = 'Не удалось выполнить запрос.'
    try {
      const payload = (await response.json()) as { error?: string }
      if (payload.error) message = payload.error
    } catch {
      // Keep the generic message when the server returned a non-JSON error.
    }
    throw new ApiError(message, response.status)
  }

  if (response.status === 204) return undefined as T
  return (await response.json()) as T
}

function formatDeadline(value: string): string {
  const deadline = new Date(value)
  if (Number.isNaN(deadline.getTime())) return value
  const today = new Date()
  const tomorrow = new Date(today)
  tomorrow.setDate(today.getDate() + 1)
  const dateKey = (date: Date) => `${date.getFullYear()}-${date.getMonth()}-${date.getDate()}`
  const time = deadline.toLocaleTimeString('ru-RU', { hour: '2-digit', minute: '2-digit' })
  if (dateKey(deadline) === dateKey(today)) return `Сегодня · ${time}`
  if (dateKey(deadline) === dateKey(tomorrow)) return `Завтра · ${time}`
  return `${deadline.toLocaleDateString('ru-RU', { day: 'numeric', month: 'long' })} · ${time}`
}

function mapTask(task: ApiTask, owners: Map<number, string> = new Map()): Task {
  return {
    id: task.id,
    title: task.title,
    subject: task.subject,
    deadline: formatDeadline(task.deadline),
    status: task.status,
    priority: priorityFromFlags(task.urgent, task.important),
    group: task.group,
    owner: owners.get(task.user_id),
  }
}

function mapMood(mood: ApiMood): MoodEntry {
  const date = new Date(mood.created_at)
  const days = ['Вс', 'Пн', 'Вт', 'Ср', 'Чт', 'Пт', 'Сб']
  return {
    id: mood.id,
    day: days[date.getDay()],
    date: date.toLocaleDateString('ru-RU', { day: '2-digit', month: '2-digit' }),
    value: mood.value === 'good' ? 18 : mood.value === 'ok' ? 13 : 8,
    mood: mood.value,
    note: mood.note,
  }
}

function mapNote(note: ApiNote): Note {
  return { id: note.id, text: note.text, allDay: note.allDay, start: note.start, end: note.end }
}

export const api = {
  authValidate: (initData: string) =>
    request<{ ok: boolean; user: { id: number; name: string } }>(
      '/auth/validate',
      { method: 'POST', body: JSON.stringify({ initData }) },
      false,
    ),
  me: () => request<ApiProfile>('/me'),
  getTasks: async () => (await request<ApiTask[]>('/tasks')).map((task) => mapTask(task)),
  createTask: async (task: Omit<Task, 'id' | 'status'> & { deadline: string }) => {
    const deadline = new Date(task.deadline)
    if (Number.isNaN(deadline.getTime())) throw new ApiError('Укажите корректный срок задачи.', 400)
    const created = await request<ApiTask>('/tasks', {
      method: 'POST',
      body: JSON.stringify({
        title: task.title,
        subject: task.subject,
        deadline: deadline.toISOString(),
        urgent: task.priority === 'urgent',
        important: task.priority !== 'later',
      }),
    })
    return mapTask(created)
  },
  setTaskStatus: async (id: Task['id'], status: TaskStatus) => {
    const updated = await request<ApiTask>(`/tasks/${encodeURIComponent(String(id))}`, {
      method: 'PATCH',
      body: JSON.stringify({ status }),
    })
    return mapTask(updated)
  },
  getMoods: async () => (await request<ApiMood[]>('/moods?days=7')).map(mapMood),
  saveMood: async (mood: MoodValue, note: string) =>
    mapMood(await request<ApiMood>('/moods', { method: 'POST', body: JSON.stringify({ mood, note }) })),
  getNotes: async () => (await request<ApiNote[]>('/notes')).map(mapNote),
  createNote: async (note: Omit<Note, 'id'>) =>
    mapNote(await request<ApiNote>('/notes', { method: 'POST', body: JSON.stringify(note) })),
  deleteNote: (id: Note['id']) =>
    request<void>(`/notes/${encodeURIComponent(String(id))}`, { method: 'DELETE' }),
  getGroupMembers: () => request<Member[]>('/group/members'),
  getGroupTasks: async (owners: Map<number, string>) =>
    (await request<ApiTask[]>('/group/tasks')).map((task) => mapTask(task, owners)),
  scanTask: (photo: File) => {
    const form = new FormData()
    form.append('photo', photo)
    return request<{ title: string; subject: string; deadline: string }>('/tasks/scan', {
      method: 'POST',
      body: form,
    })
  },
  sendFeedback: (text: string) =>
    request<{ ok: boolean }>('/feedback', { method: 'POST', body: JSON.stringify({ text }) }),
  setConsent: (value: boolean) =>
    request<{ ok: boolean }>('/consent', { method: 'POST', body: JSON.stringify({ value }) }),
  setReminders: (enabled: boolean) =>
    request<{ ok: boolean }>('/me/reminders', {
      method: 'POST',
      body: JSON.stringify({ enabled }),
    }),
}

export function apiDeadlineInput(value: string): string {
  const deadline = new Date(value)
  if (Number.isNaN(deadline.getTime())) throw new ApiError('Укажите дату и время.', 400)
  return deadline.toISOString()
}