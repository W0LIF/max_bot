import { Icon } from './ui/Icon'
import { NavProvider, TABS, useNav } from './nav'
import type { ScreenId } from './nav'
import { StoreProvider } from './store'
import { AuthOkScreen, AuthScreen, PrivacyScreen } from './screens/onboarding'
import { HomeScreen, MoodCheckScreen, MoodScreen, NoteScreen } from './screens/home'
import { AddTaskScreen, CalendarScreen, ScanScreen, TasksScreen } from './screens/planner'
import { GroupScreen, GroupTasksScreen, MapScreen } from './screens/social'
import { FeedbackScreen, HelpScreen, MenuScreen, SettingsScreen } from './screens/account'

/** Заголовок для вторичных экранов (вкладки показывают его внутри себя). */
const TITLES: Partial<Record<ScreenId, string>> = {
  'add-task': 'Добавить задачу',
  scan: 'Добавить задачу',
  note: 'Заметка',
  'mood-check': 'Отметка состояния',
  group: 'Совместный режим',
  'group-tasks': 'Группа',
  help: 'Помощь',
  feedback: 'Обратная связь',
  settings: 'Настройки',
}

function StatusBar() {
  return (
    <div className="status-bar">
      <span>9:41</span>
      <span className="status-bar__icons">
        <Icon name="signal" size={14} />
        <Icon name="wifi" size={15} />
        <Icon name="battery" size={20} />
      </span>
    </div>
  )
}

function ScreenHeader() {
  const { screen, isTab, canGoBack, back } = useNav()
  if (isTab || !canGoBack) return null

  return (
    <div className="topbar">
      <button type="button" className="icon-button" onClick={back} aria-label="Назад">
        <Icon name="back" size={18} />
      </button>
      <span className="topbar__title">{TITLES[screen] ?? 'Телескоп'}</span>
    </div>
  )
}

function BottomNav() {
  const { screen, openTab } = useNav()

  return (
    <nav className="bottom-nav">
      {TABS.map((tab) => (
        <button
          type="button"
          key={tab.id}
          className={`tab${screen === tab.id ? ' tab--active' : ''}`}
          onClick={() => openTab(tab.id)}
        >
          <Icon name={tab.icon} size={21} />
          <span className="tab__label">{tab.label}</span>
        </button>
      ))}
    </nav>
  )
}

function CurrentScreen() {
  const { screen } = useNav()

  switch (screen) {
    case 'privacy':
      return <PrivacyScreen />
    case 'auth':
      return <AuthScreen />
    case 'auth-ok':
      return <AuthOkScreen />
    case 'home':
      return <HomeScreen />
    case 'tasks':
      return <TasksScreen />
    case 'calendar':
      return <CalendarScreen />
    case 'mood':
      return <MoodScreen />
    case 'mood-check':
      return <MoodCheckScreen />
    case 'map':
      return <MapScreen />
    case 'menu':
      return <MenuScreen />
    case 'add-task':
      return <AddTaskScreen />
    case 'scan':
      return <ScanScreen />
    case 'note':
      return <NoteScreen />
    case 'group':
      return <GroupScreen />
    case 'group-tasks':
      return <GroupTasksScreen />
    case 'help':
      return <HelpScreen />
    case 'feedback':
      return <FeedbackScreen />
    case 'settings':
      return <SettingsScreen />
    default:
      return <HomeScreen />
  }
}

export function Shell() {
  const { isTab } = useNav()

  return (
    <div className="stage">
      <div className="phone">
        <StatusBar />
        <ScreenHeader />
        <CurrentScreen />
        {isTab && <BottomNav />}
      </div>
    </div>
  )
}

export default function App() {
  return (
    <StoreProvider>
      <NavProvider>
        <Shell />
      </NavProvider>
    </StoreProvider>
  )
}
