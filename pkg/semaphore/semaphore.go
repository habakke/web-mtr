package semaphore

type empty struct {}

type Semaphore struct {
	sem chan empty
}

func NewSemaphore(N int) *Semaphore {
	return &Semaphore{
		sem: make(chan empty, N),
	}
}

// acquire n resources
func (s Semaphore) P(n int) {
	e := empty{}
	for i := 0; i < n; i++ {
		s.sem <- e
	}
}

// release n resources
func (s Semaphore) V(n int) {
	for i := 0; i < n; i++ {
		<-s.sem
	}
}

func (s Semaphore) Lock() {
	s.P(1)
}

func (s Semaphore) Unlock() {
	s.V(1)
}

/* signal-wait */
func (s Semaphore) Signal() {
	s.V(1)
}

func (s Semaphore) Wait(n int) {
	s.P(n)
}

func (s Semaphore) Close() {
	close(s.sem)
}
