package dnsproxy

import (
	"net"
	"testing"
	"time"

	"github.com/miekg/dns"
)

func freeUDPAddress(t *testing.T) string {
	t.Helper()
	connection, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := connection.LocalAddr().String()
	_ = connection.Close()
	return address
}

func TestForwardAndCache(t *testing.T) {
	upstreamConnection, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	upstream := &dns.Server{PacketConn: upstreamConnection, Handler: dns.HandlerFunc(func(writer dns.ResponseWriter, request *dns.Msg) {
		response := new(dns.Msg)
		response.SetReply(request)
		response.Answer = []dns.RR{&dns.A{Hdr: dns.RR_Header{Name: request.Question[0].Name, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 30}, A: net.ParseIP("1.2.3.4")}}
		_ = writer.WriteMsg(response)
	})}
	go func() { _ = upstream.ActivateAndServe() }()
	defer upstream.Shutdown()

	listen := freeUDPAddress(t)
	service := New(Config{Enabled: true, Listen: listen, DirectUpstream: upstreamConnection.LocalAddr().String(), ProxyUpstream: upstreamConnection.LocalAddr().String(), Mode: "default_direct"})
	if err := service.Start(); err != nil {
		t.Fatal(err)
	}
	defer service.Stop()
	time.Sleep(25 * time.Millisecond)

	client := &dns.Client{Timeout: time.Second}
	for i := 0; i < 2; i++ {
		request := new(dns.Msg)
		request.SetQuestion("example.com.", dns.TypeA)
		response, _, err := client.Exchange(request, listen)
		if err != nil || len(response.Answer) != 1 {
			t.Fatalf("query %d failed: %v %#v", i, err, response)
		}
	}
	stats, logs, running := service.Snapshot()
	if !running || stats.Total != 2 || stats.Success != 2 || stats.CacheHits != 1 || len(logs) != 2 {
		t.Fatalf("unexpected snapshot: %#v, logs=%d, running=%v", stats, len(logs), running)
	}
}

func TestExceptionRouting(t *testing.T) {
	service := New(Config{Mode: "default_proxy", Exceptions: []string{"192.168.1.0/24", "10.0.0.2 # test"}})
	if service.useProxy("192.168.1.8") || service.useProxy("10.0.0.2") || !service.useProxy("10.0.0.3") {
		t.Fatal("default proxy exception routing is incorrect")
	}
	service.config.Mode = "default_direct"
	if !service.useProxy("192.168.1.8") || service.useProxy("10.0.0.3") {
		t.Fatal("default direct exception routing is incorrect")
	}
}
