package hop

import (
	"container/ring"
	"encoding/json"
	"net"
	"time"

	"github.com/habakke/web-mtr/pkg/icmp"
)

type HopStatistic struct {
	Dest           *net.IPAddr
	Timeout        time.Duration
	PID            int
	Sent           int
	TTL            int
	Target         string
	Last           icmp.ICMPReturn
	Best           icmp.ICMPReturn
	Worst          icmp.ICMPReturn
	SumElapsed     time.Duration
	Lost           int
	Packets        *ring.Ring
	RingBufferSize int
	pingSeq        int
	dnsCache       map[string]string
}

type packet struct {
	Success      bool    `json:"success"`
	ResponseTime float64 `json:"respond_ms"`
}

func New(sent, ttl int, target string, timeout time.Duration, last, best, worst icmp.ICMPReturn, lost int, sumElapsed time.Duration, packets *ring.Ring, ringBufferSize int) *HopStatistic {
	return &HopStatistic{
		Sent:           sent,
		TTL:            ttl,
		Target:         target,
		Timeout:        timeout,
		Last:           last,
		Best:           best,
		Worst:          worst,
		Lost:           lost,
		SumElapsed:     sumElapsed,
		Packets:        packets,
		RingBufferSize: ringBufferSize,
		dnsCache:       map[string]string{},
	}
}

func (s *HopStatistic) Next(srcAddr string) {
	if s.Target == "" {
		return
	}
	s.pingSeq++
	var r icmp.ICMPReturn
	if s.Dest.IP.To4() != nil {
		r, _ = icmp.SendICMP(srcAddr, s.Dest, s.Target, s.TTL, s.PID, s.Timeout, s.pingSeq)
	} else {
		r, _ = icmp.SendICMPv6(srcAddr, s.Dest, s.Target, s.TTL, s.PID, s.Timeout, s.pingSeq)
	}
	s.Packets = s.Packets.Prev()
	s.Packets.Value = r

	s.Sent++

	s.Last = r
	if !r.Success {
		s.Lost++
		return // do not count failed into statistics
	}

	s.SumElapsed = r.Elapsed + s.SumElapsed

	if s.Best.Elapsed > r.Elapsed {
		s.Best = r
	}
	if s.Worst.Elapsed < r.Elapsed {
		s.Worst = r
	}
}

func (h *HopStatistic) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Sent             int       `json:"sent"`
		Target           string    `json:"target"`
		Last             float64   `json:"last_ms"`
		Best             float64   `json:"best_ms"`
		Worst            float64   `json:"worst_ms"`
		Loss             float64   `json:"loss_percent"`
		Avg              float64   `json:"avg_ms"`
		PacketBufferSize int       `json:"-"`
		TTL              int       `json:"ttl"`
		Packets          []*packet `json:"-"`
	}{
		Sent:             h.Sent,
		TTL:              h.TTL,
		Loss:             h.Loss(),
		Target:           h.lookupAddr(true),
		PacketBufferSize: h.RingBufferSize,
		Last:             h.Last.Elapsed.Seconds() * 1000,
		Best:             h.Best.Elapsed.Seconds() * 1000,
		Worst:            h.Worst.Elapsed.Seconds() * 1000,
		Avg:              h.Avg(),
		Packets:          h.packets(),
	})
}

func (h *HopStatistic) Avg() float64 {
	avg := 0.0
	if !(h.Sent-h.Lost == 0) {
		avg = h.SumElapsed.Seconds() * 1000 / float64(h.Sent-h.Lost)
	}
	return avg
}

func (h *HopStatistic) Loss() float64 {
	return float64(h.Lost) / float64(h.Sent) * 100.0
}

func (h *HopStatistic) packets() []*packet {
	v := make([]*packet, h.RingBufferSize)
	i := 0
	h.Packets.Do(func(f interface{}) {
		if f == nil {
			v[i] = nil
			i++
			return
		}
		x := f.(icmp.ICMPReturn)
		if x.Success {
			v[i] = &packet{
				Success:      true,
				ResponseTime: x.Elapsed.Seconds() * 1000,
			}
		} else {
			v[i] = &packet{
				Success:      false,
				ResponseTime: 0.0,
			}
		}
		i++
	})
	return v
}

func (h *HopStatistic) lookupAddr(ptrLookup bool) string {
	addr := "-"
	if h.Target != "" {
		addr = h.Target
		if ptrLookup {
			if key, ok := h.dnsCache[h.Target]; ok {
				addr = key
			} else {
				names, err := net.LookupAddr(h.Target)
				if err == nil && len(names) > 0 {
					addr = names[0]
				}
			}
		}
		h.dnsCache[h.Target] = addr
	}
	return addr
}
