import { createContext, useCallback, useContext, useEffect, useMemo, useState } from 'react'
import type { ReactNode } from 'react'
import { GROUP_TASKS, INITIAL_TASKS, MEMBERS, MOOD_HISTORY, USER } from './data'
import type { Member, MoodEntry, Note, Settings, Task, TaskStatus, UserProfile } from './types'
import { sameId } from './types'
import type { MoodValue } from './ui/kit'
import { api, ApiError } from './api'
import { getInitData } from './webapp'

const STORAGE_KEY = 'telescope.state.v1'

/* История настроения всегда упорядочена по дням недели — график рисует точки
   в этом порядке, поэтому сортировка по значению его ломает. */
const WEEKDAYS = ['Пн', 'Вт', 'Ср', 'Чт', 'Пт', 'Сб', 'Вс']

const weekdayOf = (date: Date) => WEEKDAYS[(date.getDay() + 6) % 7]

const byWeekday = (moods: MoodEntry[]) => {
  const latestByDay = new Map<string, MoodEntry>()
  for (const mood of moods) {
    if (!latestByDay.has(mood.day)) latestByDay.set(mood.day, mood)
  }
  return [...latestByDay.values()].sort((a, b) => WEEKDAYS.indexOf(a.day) - WEEKDAYS.indexOf(b.day))
}

type State = {
  consent: boolean
  authorized: boolean
  tasks: Task[]
  moods: MoodEntry[]
  notes: Note[]
  members: string[]
  groupSharing: boolean
  groupMembers: Member[]
  groupTasks: Task[]
  settings: Settings
}

const DEFAULT_STATE: State = {
  consent: false,
  authorized: false,
  tasks: INITIAL_TASKS,
  moods: MOOD_HISTORY,
  notes: [],
  members: ['m1', 'm3'],
  groupSharing: false,
  groupMembers: [],
  groupTasks: [],
  settings: { notifications: true, darkTheme: false, language: 'ru' },
}

function loadState(): State {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw) return DEFAULT_STATE
    const parsed = JSON.parse(raw) as Partial<State>
    /* В localStorage могли лежать записи в перемешанном порядке — приводим к дням недели. */
    return { ...DEFAULT_STATE, ...parsed, moods: byWeekday(parsed.moods ?? DEFAULT_STATE.moods) }
  } catch {
    return DEFAULT_STATE
  }
}

type Store = {
  state: State
  user: UserProfile
  connection: 'loading' | 'online' | 'demo' | 'open-max' | 'error'
  connectionMessage: string
  error: string | null
  /* производные показатели «Твоего баланса» */
  todayTasks: Task[]
  done: number
  load: number
  progress: number
  mood: MoodValue
  acceptConsent: () => Promise<boolean>
  setScanDraft: (draft: { title: string; subject: string; deadline: string } | null) => void
  scanDraft: { title: string; subject: string; deadline: string } | null
  scanTask: (photo: File) => Promise<boolean>
  sendFeedback: (text: string) => Promise<boolean>
  dismissError: () => void
  authorize: () => void
  logout: () => void
  addTask: (task: Omit<Task, 'id' | 'status'>) => Promise<boolean>
  toggleTask: (id: Task['id']) => void
  setTaskStatus: (id: Task['id'], status: TaskStatus) => void
  saveMood: (mood: MoodValue, note: string) => Promise<boolean>
  addNote: (note: Omit<Note, 'id'>) => Promise<boolean>
  toggleMember: (id: string) => void
  setGroupSharing: (enabled: boolean) => Promise<boolean>
  setSetting: <K extends keyof Settings>(key: K, value: Settings[K]) => void
  reset: () => void
}

const StoreContext = createContext<Store | null>(null)

export function StoreProvider({ children }: { children: ReactNode }) {
  const [state, setState] = useState<State>(loadState)
  const [connection, setConnection] = useState<Store['connection']>('loading')
  const [connectionMessage, setConnectionMessage] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [user, setUser] = useState<UserProfile>(USER)
  const [scanDraft, setScanDraft] = useState<Store['scanDraft']>(null)

  useEffect(() => {
    let active = true

    const bootstrap = async () => {
      const initData = getInitData()
      const demoRequested = new URLSearchParams(window.location.search).get('demo') === '1'
      if (!initData) {
        if (import.meta.env.DEV && demoRequested) {
          if (active) {
            setState((current) => ({
              ...current,
              groupSharing: true,
              groupMembers: MEMBERS,
              groupTasks: GROUP_TASKS,
            }))
            setConnection('demo')
          }
        } else if (active) {
          setConnection('open-max')
          setConnectionMessage('Откройте приложение через кнопку в боте MAX.')
        }
        return
      }

      try {
        const auth = await api.authValidate(initData)
        const me = await api.me()
        const [tasks, moods, notes, groupMembers] = await Promise.all([
          api.getTasks(),
          api.getMoods(),
          api.getNotes(),
          api.getGroupMembers(),
        ])
        const owners = new Map(groupMembers.map((member) => [Number(member.id), member.name]))
        const groupTasks = await api.getGroupTasks(owners)
        if (!active) return

        setUser({
          name: me.name || auth.user.name || 'Студент',
          course: '',
          group: groupMembers[0]?.group || '',
        })
        setState((current) => ({
          ...current,
          consent: me.consent,
          authorized: true,
          tasks,
          moods: byWeekday(moods),
          notes,
          members: groupMembers.map((member) => String(member.id)),
          groupSharing: me.group_sharing,
          groupMembers,
          groupTasks,
          settings: { ...current.settings, notifications: me.reminders_on },
        }))
        setConnection('online')
      } catch (cause) {
        if (!active) return
        setConnection('error')
        setConnectionMessage(cause instanceof Error ? cause.message : 'Не удалось загрузить приложение.')
      }
    }

    void bootstrap()
    return () => {
      active = false
    }
  }, [])

  useEffect(() => {
    if (connection === 'online' || connection === 'demo') {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(state))
    }
  }, [connection, state])

  useEffect(() => {
    document.documentElement.dataset.theme = state.settings.darkTheme ? 'dark' : 'light'
  }, [state.settings.darkTheme])

  const acceptConsent = useCallback(async () => {
    try {
      if (connection === 'online') await api.setConsent(true)
      setState((s) => ({ ...s, consent: true }))
      return true
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : 'Не удалось сохранить согласие.')
      return false
    }
  }, [connection])

  const authorize = useCallback(() => setState((s) => ({ ...s, authorized: true })), [])
  const logout = useCallback(
    () => setState((s) => ({ ...s, authorized: false, consent: false })),
    [],
  )

  const addTask = useCallback(async (task: Omit<Task, 'id' | 'status'>) => {
    try {
      const created = connection === 'online'
        ? await api.createTask(task)
        : { ...task, id: `t${Date.now()}`, status: 'new' as const }
      setState((s) => ({ ...s, tasks: [created, ...s.tasks] }))
      return true
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : 'Не удалось сохранить задачу.')
      return false
    }
  }, [connection])

  const toggleTask = useCallback((id: Task['id']) => {
    const task = state.tasks.find((item) => sameId(item.id, id))
    if (!task) return
    const status = task.status === 'done' ? 'new' : 'done'
    void (async () => {
      try {
        const updated: Task = connection === 'online' ? await api.setTaskStatus(id, status) : { ...task, status }
        setState((s) => ({ ...s, tasks: s.tasks.map((item) => sameId(item.id, id) ? updated : item) }))
      } catch (cause) {
        setError(cause instanceof Error ? cause.message : 'Не удалось изменить задачу.')
      }
    })()
  }, [connection, state.tasks])

  const setTaskStatus = useCallback((id: Task['id'], status: TaskStatus) => {
    void (async () => {
      try {
        const task = state.tasks.find((item) => sameId(item.id, id))
        if (!task) return
        const updated: Task = connection === 'online' ? await api.setTaskStatus(id, status) : { ...task, status }
        setState((s) => ({ ...s, tasks: s.tasks.map((item) => sameId(item.id, id) ? updated : item) }))
      } catch (cause) {
        setError(cause instanceof Error ? cause.message : 'Не удалось изменить задачу.')
      }
    })()
  }, [connection, state.tasks])

  const saveMood = useCallback(async (mood: MoodValue, note: string) => {
    const today = new Date()
    const day = weekdayOf(today)
    try {
      const entry: MoodEntry = connection === 'online'
        ? await api.saveMood(mood, note)
        : {
            id: `d${today.getTime()}`,
            day,
            date: today.toLocaleDateString('ru-RU').slice(0, 5),
            value: mood === 'good' ? 18 : mood === 'ok' ? 13 : 8,
            mood,
            note: note || undefined,
          }
      setState((s) => {
        const rest = s.moods.filter((m) => m.day !== day)
        return { ...s, moods: byWeekday([{ ...entry, day }, ...rest]) }
      })
      return true
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : 'Не удалось сохранить настроение.')
      return false
    }
  }, [connection])

  const addNote = useCallback(async (note: Omit<Note, 'id'>) => {
    try {
      const today = new Date()
      const localDate = `${today.getFullYear()}-${String(today.getMonth() + 1).padStart(2, '0')}-${String(today.getDate()).padStart(2, '0')}`
      const dateTime = (time: string, end: boolean) => {
        const value = note.allDay ? `${localDate}T${end ? '23:59' : '00:00'}` : `${localDate}T${time || '00:00'}`
        return new Date(value).toISOString()
      }
      const payload = { ...note, start: dateTime(note.start, false), end: dateTime(note.end, true) }
      const created = connection === 'online' ? await api.createNote(payload) : { ...note, id: `n${Date.now()}` }
      setState((s) => ({ ...s, notes: [created, ...s.notes] }))
      return true
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : 'Не удалось сохранить заметку.')
      return false
    }
  }, [connection])

  const toggleMember = useCallback((id: string) => {
    setState((s) => ({
      ...s,
      members: s.members.includes(id) ? s.members.filter((m) => m !== id) : [...s.members, id],
    }))
  }, [])

  const setGroupSharing = useCallback(async (enabled: boolean) => {
    try {
      if (connection === 'online') {
        await api.setGroupSharing(enabled)
        if (!enabled) {
          setState((s) => ({
            ...s,
            groupSharing: false,
            members: [],
            groupMembers: [],
            groupTasks: [],
          }))
          return true
        }

        const groupMembers = await api.getGroupMembers()
        const owners = new Map(groupMembers.map((member) => [Number(member.id), member.name]))
        const groupTasks = await api.getGroupTasks(owners)
        setState((s) => ({
          ...s,
          groupSharing: true,
          members: groupMembers.map((member) => String(member.id)),
          groupMembers,
          groupTasks,
        }))
        return true
      }

      if (connection === 'demo') {
        setState((s) => ({
          ...s,
          groupSharing: enabled,
          members: enabled ? MEMBERS.map((member) => String(member.id)) : [],
          groupMembers: enabled ? MEMBERS : [],
          groupTasks: enabled ? GROUP_TASKS : [],
        }))
        return true
      }

      throw new Error('Групповой доступ доступен после входа через MAX.')
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : 'Не удалось изменить общий доступ.')
      return false
    }
  }, [connection])

  const setSetting = useCallback(<K extends keyof Settings>(key: K, value: Settings[K]) => {
    void (async () => {
      try {
        if (key === 'notifications' && connection === 'online') await api.setReminders(value as boolean)
        setState((s) => ({ ...s, settings: { ...s.settings, [key]: value } }))
      } catch (cause) {
        setError(cause instanceof Error ? cause.message : 'Не удалось сохранить настройку.')
      }
    })()
  }, [connection])

  const reset = useCallback(() => setState(DEFAULT_STATE), [])
  const scanTask = useCallback(async (photo: File) => {
    try {
      const scanned = connection === 'online'
        ? await api.scanTask(photo)
        : { title: photo.name.replace(/\.[^.]+$/, '') || 'Новая задача', subject: 'Общее', deadline: new Date(Date.now() + 86400000).toISOString() }
      const date = new Date(scanned.deadline)
      const localDeadline = new Date(date.getTime() - date.getTimezoneOffset() * 60000).toISOString().slice(0, 16)
      setScanDraft({ ...scanned, deadline: localDeadline })
      return true
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : 'Не удалось обработать фотографию.')
      return false
    }
  }, [connection])
  const sendFeedback = useCallback(async (text: string) => {
    try {
      if (connection === 'online') await api.sendFeedback(text)
      return true
    } catch (cause) {
      setError(cause instanceof ApiError ? cause.message : 'Не удалось отправить сообщение.')
      return false
    }
  }, [connection])
  const dismissError = useCallback(() => setError(null), [])

  const derived = useMemo(() => {
    const todayTasks = state.tasks.filter((t) => t.status !== 'done').slice(0, 4)
    const done = state.tasks.filter((t) => t.status === 'done').length
    const progress = state.tasks.length ? Math.round((done / state.tasks.length) * 100) : 0
    const last = state.moods[state.moods.length - 1]
    const todayEntry = state.moods.find((m) => m.day === weekdayOf(new Date()))
    return {
      todayTasks,
      done,
      progress,
      load: Math.min(3, todayTasks.length > 2 ? 3 : Math.max(1, todayTasks.length)),
      mood: (todayEntry?.mood ?? last?.mood ?? 'good') as MoodValue,
    }
  }, [state.tasks, state.moods])

  const value: Store = {
    state,
    user,
    connection,
    connectionMessage,
    error,
    ...derived,
    acceptConsent,
    setScanDraft,
    scanDraft,
    scanTask,
    sendFeedback,
    dismissError,
    authorize,
    logout,
    addTask,
    toggleTask,
    setTaskStatus,
    saveMood,
    addNote,
    toggleMember,
    setGroupSharing,
    setSetting,
    reset,
  }

  return <StoreContext.Provider value={value}>{children}</StoreContext.Provider>
}

// eslint-disable-next-line react-refresh/only-export-components
export function useStore(): Store {
  const store = useContext(StoreContext)
  if (!store) throw new Error('useStore должен вызываться внутри StoreProvider')
  return store
}
