package service

import (
	"log"
	"sync"
	"time"

	"cmsingbox.local/cmsingbox/internal/storage"
)

// Scheduler 定时任务调度器
type Scheduler struct {
	store      *storage.JSONStore
	subService *SubscriptionService
	onUpdate   func() error // 订阅更新后的回调

	stopCh   chan struct{}
	running  bool
	interval time.Duration
	mu       sync.Mutex
}

// NewScheduler 创建调度器
func NewScheduler(store *storage.JSONStore, subService *SubscriptionService) *Scheduler {
	return &Scheduler{
		store:      store,
		subService: subService,
		stopCh:     make(chan struct{}),
	}
}

// SetUpdateCallback 设置更新回调
func (s *Scheduler) SetUpdateCallback(callback func() error) {
	s.onUpdate = callback
}

// Start 启动调度器
func (s *Scheduler) Start() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return
	}

	settings := s.store.GetSettings()
	if settings.SubscriptionInterval <= 0 {
		log.Println("[Scheduler] 定时更新已禁用")
		return
	}

	s.interval = time.Duration(settings.SubscriptionInterval) * time.Minute
	s.running = true
	s.stopCh = make(chan struct{})

	go s.run()
	log.Printf("[Scheduler] 已启动，更新间隔: %v\n", s.interval)
}

// Stop 停止调度器
func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return
	}

	close(s.stopCh)
	s.running = false
	log.Println("[Scheduler] 已停止")
}

// Restart 重启调度器（更新配置后调用）
func (s *Scheduler) Restart() {
	s.Stop()
	s.Start()
}

// IsRunning 检查是否运行中
func (s *Scheduler) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}

// run 运行定时任务
func (s *Scheduler) run() {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.updateSubscriptions()
		}
	}
}

// updateSubscriptions 更新所有订阅
func (s *Scheduler) updateSubscriptions() {
	log.Println("[Scheduler] 开始自动更新订阅...")

	if err := s.subService.RefreshAll(); err != nil {
		log.Printf("[Scheduler] 更新订阅失败: %v\n", err)
		return
	}

	log.Println("[Scheduler] 订阅更新完成")

	// 调用更新回调（自动应用配置）
	if s.onUpdate != nil {
		if err := s.onUpdate(); err != nil {
			log.Printf("[Scheduler] 自动应用配置失败: %v\n", err)
		} else {
			log.Println("[Scheduler] 配置已自动应用")
		}
	}
}

// GetNextUpdateTime 获取下次更新时间
func (s *Scheduler) GetNextUpdateTime() *time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return nil
	}

	next := time.Now().Add(s.interval)
	return &next
}

// GetInterval 获取更新间隔
func (s *Scheduler) GetInterval() time.Duration {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.interval
}
