package main

import (
	"fmt"
	"log"
	"net"
	"strconv"
	"time"

	"github.com/LeoMarche/cg-dn-server/pkg/records"
	"github.com/enriquebris/goconcurrentqueue"
	"github.com/miekg/dns"
)

type Request struct {
	w   *dns.ResponseWriter
	msg *dns.Msg
	t   uint16
}

var cachedRecordsA = records.NewRecordsList()
var cachedRecordsAAAA = records.NewRecordsList()
var dnsServers = []string{"1.1.1.1", "8.8.8.8", "1.0.0.1", "8.8.4.4"}
var q = goconcurrentqueue.NewFixedFIFO(1000)

type handler struct{}

func (this *handler) ServeDNS(w dns.ResponseWriter, r *dns.Msg) {
	msg := dns.Msg{}
	msg.SetReply(r)
	q.Enqueue(Request{&w, &msg, r.Question[0].Qtype})
}

func main() {
	stop := make(chan bool)
	for i := 0; i < 1000; i++ {
		go Resolver(stop, q)
	}
	srv := &dns.Server{Addr: ":" + strconv.Itoa(8090), Net: "udp"}
	srv.Handler = &handler{}
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("Failed to set udp listener %s\n", err.Error())
	}
}

func Resolver(stop chan bool, q *goconcurrentqueue.FixedFIFO) {
	for {
		select {
		case <-stop:
			return
		default:
			value, _ := q.DequeueOrWaitForNextElement()
			if value != nil {
				v := value.(Request)
				msg := v.msg
				w := *(v.w)
				t := v.t
				msg.Authoritative = true
				msg.RecursionAvailable = true
				domain := msg.Question[0].Name
				switch t {
				case dns.TypeA:
					addresses, err := ResolveA(cachedRecordsA, domain)
					if err == nil {
						for _, ip := range addresses {
							msg.Answer = append(msg.Answer, &dns.A{
								Hdr: dns.RR_Header{Name: domain, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 60},
								A:   net.ParseIP(ip),
							})
						}
					}
					w.WriteMsg(msg)
				case dns.TypeAAAA:
					addresses, err := ResolveAAAA(cachedRecordsAAAA, domain)
					if err == nil {
						for _, ip := range addresses {
							msg.Answer = append(msg.Answer, &dns.AAAA{
								Hdr:  dns.RR_Header{Name: domain, Rrtype: dns.TypeAAAA, Class: dns.ClassINET, Ttl: 60},
								AAAA: net.ParseIP(ip),
							})
						}
					}
					w.WriteMsg(msg)
				}
			}
		}
	}
}

func ResolveA(rl *records.RecordsList, dn string) ([]string, error) {
	c := new(dns.Client)
	t := time.Now().Second()
	ok, ips := rl.Read(dn, t)
	if ok {
		return ips, nil
	}
	m := dns.Msg{}
	m.SetQuestion(dn, dns.TypeA)
	req, _, err := c.Exchange(&m, dnsServers[0]+":53")

	if err != nil {
		return nil, err
	} else if len(req.Answer) == 0 {
		return nil, fmt.Errorf("No records found")
	} else {
		ips := []string{}
		var ttl int
		for _, ans := range req.Answer {
			if a, ok := ans.(*dns.A); ok {
				ips = append(ips, a.A.String())
				ttl = int(ans.Header().Ttl)
			}
		}
		go rl.Append(dn, ips, t+ttl)
		return ips, nil
	}
}

func ResolveAAAA(rl *records.RecordsList, dn string) ([]string, error) {
	c := new(dns.Client)
	t := time.Now().Second()
	ok, ips := rl.Read(dn, t)
	if ok {
		return ips, nil
	}
	m := dns.Msg{}
	m.SetQuestion(dn, dns.TypeAAAA)
	req, _, err := c.Exchange(&m, dnsServers[0]+":53")

	if err != nil {
		return nil, err
	} else if len(req.Answer) == 0 {
		return nil, fmt.Errorf("No records found")
	} else {
		ips := []string{}
		var ttl int
		for _, ans := range req.Answer {
			if a, ok := ans.(*dns.AAAA); ok {
				ips = append(ips, a.AAAA.String())
				ttl = int(ans.Header().Ttl)
			}
		}
		go rl.Append(dn, ips, t+ttl)
		return ips, nil
	}
}

func ResolveCNAME() {}

func ResolveMX() {}

func ResolveSOA() {}
