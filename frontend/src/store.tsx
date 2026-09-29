import { createContext, useCallback, useContext, useEffect, useMemo, useState } from 'react'
import type { ReactNode } from 'react'
import { INITIAL_TASKS, MOOD_HISTORY, USER } from './data'
import type { MoodEntry, Note, Task, TaskStatus } from './data'
import type { MoodValue } from './ui/kit'

const STORAGE_KEY = 'telescope.state.v1'

type Settings = {
  notifications: boolean
  darkTheme: boolean
  language: 'ru' | 'en'
}

type State = {
  consent: boolean
  authorized: boolean
  tasks: Task[]
  moods: MoodEntry[]
  notes: Note[]
  members: string[]
  settings: Settings
}

const DEFAULT_STATE: State = {
  consent: false,
  authorized: false,
  tasks: INITIAL_TASKS,
  moods: MOOD_HISTORY,
  notes: [],
  members: ['m1', 'm3'],
  settings: { notifications: true, darkTheme: false, language: 'ru' },
}

function loadState(): State {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw) return DEFAULT_STATE
    return { ...DEFAULT_STATE, ...(JSON.parse(raw) as Partial<State>) }
  } catch {
    return DEFAULT_STATE
  }
}

type Store = {
  state: State
  user: typeof USER
  /* производные показатели «Твоего баланса» */
  todayTasks: Task[]
  done: number
  load: number
  progress: number
  mood: MoodValue
  acceptConsent: () => void
  authorize: () => void
  logout: () => void
  addTask: (task: Omit<Task, 'id' | 'status'>) => void
  toggleTask: (id: string) => void
  setTaskStatus: (id: string, status: TaskStatus) => void
  saveMood: (mood: MoodValue, note: string) => void
  addNote: (note: Omit<Note, 'id'>) => void
  toggleMember: (id: string) => void
  setSetting: <K extends keyof Settings>(key: K, value: Settings[K]) => void
  reset: () => void
}

const StoreContext = createContext<Store | null>(null)

export function StoreProvider({ children }: { children: ReactNode }) {
  const [state, setState] = useState<State>(loadState)

  useEffect(() => {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(state))
  }, [state])

  const acceptConsent = useCallback(() => setState((s) => ({ ...s, consent: true })), [])
  const authorize = useCallback(() => setState((s) => ({ ...s, authorized: true })), [])
  const logout = useCallback(
    () => setState((s) => ({ ...s, authorized: false, consent: false })),
    [],
  )

  const addTask = useCallback((task: Omit<Task, 'id' | 'status'>) => {
    setState((s) => ({
      ...s,
      tasks: [{ ...task, id: `t${Date.now()}`, status: 'new' }, ...s.tasks],
    }))
  }, [])

  const toggleTask = useCallback((id: string) => {
    setState((s) => ({
      ...s,
      tasks: s.tasks.map((t) =>
        t.id === id ? { ...t, status: t.status === 'done' ? 'in_progress' : 'done' } : t,
      ),
    }))
  }, [])

  const setTaskStatus = useCallback((id: string, status: TaskStatus) => {
    setState((s) => ({ ...s, tasks: s.tasks.map((t) => (t.id === id ? { ...t, status } : t)) }))
  }, [])

  const saveMood = useCallback((mood: MoodValue, note: string) => {
    setState((s) => {
      const today = new Date()
      const weekdays = ['Пн', 'Вт', 'Ср', 'Чт', 'Пт', 'Сб', 'Вс']
      const index = (today.getDay() + 6) % 7
      const entry: MoodEntry = {
        id: `d${today.getTime()}`,
        day: weekdays[index],
        date: today.toLocaleDateString('ru-RU').slice(0, 5),
        value: mood === 'good' ? 18 : mood === 'ok' ? 13 : 8,
        mood,
        note: note || undefined,
      }
      const rest = s.moods.filter((m) => m.day !== entry.day)
      return { ...s, moods: [...rest, entry].sort((a, b) => a.value - b.value) }
    })
  }, [])

  const addNote = useCallback((note: Omit<Note, 'id'>) => {
    setState((s) => ({ ...s, notes: [{ ...note, id: `n${Date.now()}` }, ...s.notes] }))
  }, [])

  const toggleMember = useCallback((id: string) => {
    setState((s) => ({
      ...s,
      members: s.members.includes(id) ? s.members.filter((m) => m !== id) : [...s.members, id],
    }))
  }, [])

  const setSetting = useCallback(<K extends keyof Settings>(key: K, value: Settings[K]) => {
    setState((s) => ({ ...s, settings: { ...s.settings, [key]: value } }))
  }, [])

  const reset = useCallback(() => setState(DEFAULT_STATE), [])

  const derived = useMemo(() => {
    const todayTasks = state.tasks.filter((t) => t.status !== 'done').slice(0, 4)
    const done = state.tasks.filter((t) => t.status === 'done').length
    const progress = state.tasks.length ? Math.round((done / state.tasks.length) * 100) : 0
    const last = state.moods[state.moods.length - 1]
    return {
      todayTasks,
      done,
      progress,
      load: Math.min(3, todayTasks.length > 2 ? 3 : Math.max(1, todayTasks.length)),
      mood: (last?.mood ?? 'good') as MoodValue,
    }
  }, [state.tasks, state.moods])

  const value: Store = {
    state,
    user: USER,
    ...derived,
    acceptConsent,
    authorize,
    logout,
    addTask,
    toggleTask,
    setTaskStatus,
    saveMood,
    addNote,
    toggleMember,
    setSetting,
    reset,
  }

  return <StoreContext.Provider value={value}>{children}</StoreContext.Provider>
}

export function useStore(): Store {
  const store = useContext(StoreContext)
  if (!store) throw new Error('useStore должен вызываться внутри StoreProvider')
  return store
}
