package dnsproxy

import (
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/miekg/dns"
)

type Config struct {
	Enabled        bool
	Listen         string
	ProxyUpstream  string
	DirectUpstream string
	Mode           string
	Exceptions     []string
}

type QueryLog struct {
	Time       string  `json:"time"`
	Domain     string  `json:"domain"`
	Type       string  `json:"type"`
	Upstream   string  `json:"upstream"`
	SourceIP   string  `json:"source_ip"`
	Result     string  `json:"result"`
	DurationMS float64 `json:"duration_ms"`
	Cached     bool    `json:"cached"`
}

type Stats struct {
	Total             uint64  `json:"total"`
	Success           uint64  `json:"success"`
	Failed            uint64  `json:"failed"`
	CacheHits         uint64  `json:"cache_hits"`
	AverageResponseMS float64 `json:"average_response_ms"`
}

type cacheEntry struct {
	message *dns.Msg
	expires time.Time
}

type Service struct {
	mu            sync.RWMutex
	config        Config
	udp           *dns.Server
	tcp           *dns.Server
	cache         map[string]cacheEntry
	logs          []QueryLog
	stats         Stats
	totalDuration float64
}

func New(config Config) *Service {
	return &Service{config: config, cache: make(map[string]cacheEntry), logs: make([]QueryLog, 0, 1000)}
}

func (s *Service) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.config.Enabled || s.udp != nil {
		return nil
	}
	if _, _, err := net.SplitHostPort(s.config.Listen); err != nil {
		return fmt.Errorf("无效 DNS 监听地址: %w", err)
	}
	handler := dns.HandlerFunc(s.handle)
	packetConnection, err := net.ListenPacket("udp", s.config.Listen)
	if err != nil {
		return fmt.Errorf("监听 DNS UDP 失败: %w", err)
	}
	streamListener, err := net.Listen("tcp", s.config.Listen)
	if err != nil {
		_ = packetConnection.Close()
		return fmt.Errorf("监听 DNS TCP 失败: %w", err)
	}
	s.udp = &dns.Server{PacketConn: packetConnection, Handler: handler}
	s.tcp = &dns.Server{Listener: streamListener, Handler: handler}
	udp, tcp := s.udp, s.tcp
	go func() { _ = udp.ActivateAndServe() }()
	go func() { _ = tcp.ActivateAndServe() }()
	return nil
}

func (s *Service) Stop() {
	s.mu.Lock()
	udp, tcp := s.udp, s.tcp
	s.udp, s.tcp = nil, nil
	s.mu.Unlock()
	if udp != nil {
		_ = udp.Shutdown()
	}
	if tcp != nil {
		_ = tcp.Shutdown()
	}
}

func (s *Service) Update(config Config) error {
	s.Stop()
	s.mu.Lock()
	s.config = config
	s.cache = make(map[string]cacheEntry)
	s.mu.Unlock()
	return s.Start()
}

func (s *Service) Snapshot() (Stats, []QueryLog, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.stats, append([]QueryLog(nil), s.logs...), s.udp != nil || s.tcp != nil
}

func (s *Service) useProxy(source string) bool {
	ip := net.ParseIP(source)
	s.mu.RLock()
	entries, mode := append([]string(nil), s.config.Exceptions...), s.config.Mode
	s.mu.RUnlock()
	exception := false
	for _, entry := range entries {
		entry = strings.TrimSpace(strings.Split(entry, "#")[0])
		if candidate := net.ParseIP(entry); candidate != nil && candidate.Equal(ip) {
			exception = true
			break
		}
		if _, cidr, err := net.ParseCIDR(entry); err == nil && cidr.Contains(ip) {
			exception = true
			break
		}
	}
	if mode == "default_direct" {
		return exception
	}
	return !exception
}

func (s *Service) handle(writer dns.ResponseWriter, request *dns.Msg) {
	started := time.Now()
	source, _, _ := net.SplitHostPort(writer.RemoteAddr().String())
	key := cacheKey(request)
	s.mu.Lock()
	if item, ok := s.cache[key]; ok && time.Now().Before(item.expires) {
		response := item.message.Copy()
		response.Id = request.Id
		s.stats.Total++
		s.stats.Success++
		s.stats.CacheHits++
		s.logs = s.appendLog(QueryLog{Time: time.Now().Format(time.RFC3339), Domain: questionName(request), Type: questionType(request), Upstream: "cache", SourceIP: source, Result: "success", Cached: true})
		s.mu.Unlock()
		_ = writer.WriteMsg(response)
		return
	}
	config := s.config
	s.mu.Unlock()

	upstream := config.DirectUpstream
	if s.useProxy(source) {
		upstream = config.ProxyUpstream
	}
	client := &dns.Client{Net: "udp", Timeout: 5 * time.Second}
	response, _, err := client.Exchange(request, upstream)
	duration := float64(time.Since(started).Microseconds()) / 1000
	result := "success"
	if err != nil {
		result = "failed"
		response = new(dns.Msg)
		response.SetRcode(request, dns.RcodeServerFailure)
	}
	s.mu.Lock()
	s.stats.Total++
	s.totalDuration += duration
	s.stats.AverageResponseMS = s.totalDuration / float64(s.stats.Total)
	if err == nil {
		s.stats.Success++
		if ttl := minimumTTL(response); ttl > 0 {
			s.cache[key] = cacheEntry{message: response.Copy(), expires: time.Now().Add(time.Duration(ttl) * time.Second)}
		}
	} else {
		s.stats.Failed++
	}
	s.logs = s.appendLog(QueryLog{Time: time.Now().Format(time.RFC3339), Domain: questionName(request), Type: questionType(request), Upstream: upstream, SourceIP: source, DurationMS: duration, Result: result})
	s.mu.Unlock()
	_ = writer.WriteMsg(response)
}

func (s *Service) appendLog(item QueryLog) []QueryLog {
	logs := append([]QueryLog{item}, s.logs...)
	if len(logs) > 1000 {
		logs = logs[:1000]
	}
	return logs
}

func cacheKey(message *dns.Msg) string {
	if len(message.Question) == 0 {
		return ""
	}
	q := message.Question[0]
	return strings.ToLower(q.Name) + fmt.Sprintf("/%d", q.Qtype)
}
func questionName(message *dns.Msg) string {
	if len(message.Question) > 0 {
		return strings.TrimSuffix(message.Question[0].Name, ".")
	}
	return ""
}
func questionType(message *dns.Msg) string {
	if len(message.Question) > 0 {
		return dns.TypeToString[message.Question[0].Qtype]
	}
	return ""
}
func minimumTTL(message *dns.Msg) uint32 {
	var ttl uint32
	for _, answer := range message.Answer {
		value := answer.Header().Ttl
		if ttl == 0 || value < ttl {
			ttl = value
		}
	}
	return ttl
}
