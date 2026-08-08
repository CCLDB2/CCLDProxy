package main

import "sync"

// limiter controla el maximo de conexiones simultaneas por IP.
// maxPerIP == 0 significa sin limite.
type limiter struct {
	max     int
	mu      sync.Mutex
	counts  map[string]int
	logger  *logger
}

func newLimiter(max int, logger *logger) *limiter {
	return &limiter{max: max, counts: make(map[string]int), logger: logger}
}

// acquire reserva una conexion para la IP. false si supera el limite.
func (l *limiter) acquire(ip string) bool {
	if l.max <= 0 {
		return true
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	c := l.counts[ip]
	if c >= l.max {
		return false
	}
	l.counts[ip] = c + 1
	return true
}

// release libera una conexion de la IP.
func (l *limiter) release(ip string) {
	if l.max <= 0 {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if c := l.counts[ip]; c > 1 {
		l.counts[ip] = c - 1
	} else {
		delete(l.counts, ip)
	}
}
