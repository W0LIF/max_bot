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
	sent, err := DispatchDue(ctx, s.store, time.Now().Add(s.before), func(ctx context.Context, task storage.Task) error {
		select {
		case s.out <- task:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	})
	if err != nil {
		s.log.Error("не удалось обработать напоминания", "err", err)
	}
	if sent > 0 {
		s.log.Info("напоминания поставлены в очередь", "count", sent)
	}
}
