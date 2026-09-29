import { createContext, useCallback, useContext, useMemo, useState } from 'react'
import type { ReactNode } from 'react'
import type { IconName } from './ui/Icon'

/** Экраны макета: 1-3 онбординг, вкладки, вторичные экраны. */
export type ScreenId =
  | 'privacy'
  | 'auth'
  | 'auth-ok'
  | 'home'
  | 'tasks'
  | 'calendar'
  | 'mood'
  | 'map'
  | 'menu'
  | 'add-task'
  | 'edit-task'
  | 'scan'
  | 'mood-check'
  | 'note'
  | 'group'
  | 'group-tasks'
  | 'help'
  | 'feedback'
  | 'settings'

// eslint-disable-next-line react-refresh/only-export-components
export const TABS: { id: ScreenId; label: string; icon: IconName }[] = [
  { id: 'home', label: 'Главная', icon: 'home' },
  { id: 'calendar', label: 'Даты', icon: 'calendar' },
  { id: 'tasks', label: 'Задачи', icon: 'tasks' },
  { id: 'mood', label: 'Баланс', icon: 'mood' },
  { id: 'map', label: 'Поход', icon: 'map' },
  { id: 'menu', label: 'Ещё', icon: 'menu' },
]

type Nav = {
  screen: ScreenId
  isTab: boolean
  canGoBack: boolean
  push: (screen: ScreenId) => void
  openTab: (screen: ScreenId) => void
  replace: (screen: ScreenId) => void
  back: () => void
}

const NavContext = createContext<Nav | null>(null)

export function NavProvider({ children, initial = initialScreen() }: { children: ReactNode; initial?: ScreenId }) {
  const [stack, setStack] = useState<ScreenId[]>([initial])

  const push = useCallback((screen: ScreenId) => setStack((s) => [...s, screen]), [])
  const replace = useCallback((screen: ScreenId) => setStack((s) => [...s.slice(0, -1), screen]), [])
  const openTab = useCallback((screen: ScreenId) => setStack([screen]), [])
  const back = useCallback(() => setStack((s) => (s.length > 1 ? s.slice(0, -1) : s)), [])

  const screen = stack[stack.length - 1]
  const value = useMemo<Nav>(
    () => ({
      screen,
      isTab: TABS.some((t) => t.id === screen),
      canGoBack: stack.length > 1,
      push,
      openTab,
      replace,
      back,
    }),
    [screen, stack.length, push, openTab, replace, back],
  )

  return <NavContext.Provider value={value}>{children}</NavContext.Provider>
}

function initialScreen(): ScreenId {
  const params = new URLSearchParams(window.location.search)
  return params.get('from') === 'bot' || params.get('demo') === '1' ? 'home' : 'privacy'
}

// eslint-disable-next-line react-refresh/only-export-components
export function useNav(): Nav {
  const nav = useContext(NavContext)
  if (!nav) throw new Error('useNav должен вызываться внутри NavProvider')
  return nav
}
