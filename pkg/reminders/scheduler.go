package reminders

import (
	"context"
	"log/slog"
	"time"

	"max_bot_api/pkg/storage"
)

type Scheduler struct {
	store    storage.Store
	out      chan<- storage.Task
	interval time.Duration
	before   time.Duration
	log      *slog.Logger
}

func NewScheduler(
	store storage.Store,
	out chan<- storage.Task,
	interval time.Duration,
	before time.Duration,
	log *slog.Logger,
) *Scheduler {
	if log == nil {
		log = slog.Default()
	}
	return &Scheduler{
		store:    store,
		out:      out,
		interval: interval,
		before:   before,
		log:      log,
	}
}

func (s *Scheduler) Run(ctx context.Context) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	s.log.Info("шедулер напоминаний запущен",
		"interval", s.interval,
		"before", s.before,
	)

	for {
		select {
		case <-ctx.Done():
			s.log.Info("шедулер напоминаний остановлен")
			return
		case <-ticker.C:
			s.tick(ctx)
		}
	}
}

func (s *Scheduler) tick(ctx context.Context) {
	before := time.Now().Add(s.before)

	tasks, err := s.store.GetTasksDueBefore(ctx, before)
	if err != nil {
		s.log.Error("не удалось получить задачи с близким дедлайном", "err", err)
		return
	}

	if len(tasks) == 0 {
		return
	}

	s.log.Info("найдены задачи для напоминания", "count", len(tasks))

	for _, task := range tasks {
		if err := s.store.SetTaskReminderSent(ctx, task.ID); err != nil {
			s.log.Error("не удалось пометить напоминание отправленным",
				"task_id", task.ID,
				"err", err,
			)
			continue
		}

		select {
		case s.out <- task:
		case <-ctx.Done():
			return
		}
	}
}
