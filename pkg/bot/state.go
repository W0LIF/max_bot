package bot

import "sync"

// State — состояние диалога пользователя.
type State int

const (
	StateIdle State = iota

	// Добавление новой задачи
	StateAwaitTaskTitle
	StateAwaitTaskDeadline
	StateAwaitTaskPriority

	// Редактирование существующей
	StateAwaitEditTitle
	StateAwaitEditDeadline
	StateAwaitEditPriority
)

// draft — то, что пользователь вводит по шагам, пока не дошёл до сохранения.
// EditID == 0 → создаём новую задачу; EditID > 0 → правим задачу с этим ID.
type draft struct {
	EditID   int64
	Title    string
	Deadline string
	Priority int
}

// FSM хранит состояния и черновики по каждому userID.
//
// ВАЖНО: это память процесса. Для long-polling работает.
// На Vercel serverless состояние между запросами не сохраняется —
// это общая задача №4 из распределения, решать вместе с Разр1.
type FSM struct {
	mu     sync.Mutex
	states map[int64]State
	drafts map[int64]*draft
}

func NewFSM() *FSM {
	return &FSM{
		states: make(map[int64]State),
		drafts: make(map[int64]*draft),
	}
}

func (f *FSM) Get(userID int64) State {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.states[userID]
}

func (f *FSM) Set(userID int64, s State) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.states[userID] = s
}

// Draft возвращает черновик пользователя, создавая его при необходимости.
// Возвращается указатель — меняй поля напрямую.
func (f *FSM) Draft(userID int64) *draft {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.drafts[userID] == nil {
		f.drafts[userID] = &draft{}
	}
	return f.drafts[userID]
}

// Reset — вернуть пользователя в Idle и очистить черновик.
// Вызывать после сохранения/отмены сценария.
func (f *FSM) Reset(userID int64) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.states[userID] = StateIdle
	delete(f.drafts, userID)
}
