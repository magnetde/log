package log

import (
	"runtime"
	"sync"
)

// queue ist eine Datenstruktur, die Jobs in einer Goroutine asynchron ausführt.
//
// Die Queue ist nicht Thread-sicher. Wenn die Queue in verschiedenen Goroutinen ausgeführt werden soll,
// müssen geeignete Maßnahmen noch hinzugefügt werden.
type queue struct {
	Handler          func(interface{})
	ConcurrencyLimit int
	push             chan interface{}
	pop              chan struct{}
	suspend          chan bool
	suspended        bool
	stop             chan struct{}
	stopped          bool
	buffer           []interface{}
	count            int
	wg               sync.WaitGroup
}

// Handler ist die Funktion, die einzelne Jobs ausführt.
// Dabei ist der Parameter das Objekt, dass zur Queue hinzugefügt wurde.
type handler func(interface{})

// NewQueue erstellt eine neue Queue. Als Parameter wird dabei die Handler-Funktion
// und die maximale Anzahl von parallelen Jobs benötigt.
func newQueue(handler handler, concurrencyLimit int) *queue {
	q := &queue{
		Handler:          handler,
		ConcurrencyLimit: concurrencyLimit,
		push:             make(chan interface{}),
		pop:              make(chan struct{}),
		suspend:          make(chan bool),
		stop:             make(chan struct{}),
	}

	go q.run()
	runtime.SetFinalizer(q, (*queue).stopQueue)
	return q
}

// pushJob fügt einen neuen Job zu der Queue hinzu.
func (q *queue) pushJob(val interface{}) {
	q.push <- val
}

// stopQueue stoppt die Queue. Anschließend können keine neuen Job aufgenommen.
// Jobs, die noch auf der Queue existieren werden dabei noch im Hintergrund abgearbeitet.
func (q *queue) stopQueue() {
	q.stop <- struct{}{}
	runtime.SetFinalizer(q, nil)
}

// wait wartet solange, bis keine Jobs mehr in der Queue vorhanden sind.
func (q *queue) wait() {
	q.wg.Wait()
}

// len gibt die Anzahl der Jobs in der Queue zurück.
func (q *queue) len() (_, _ int) {
	return q.count, len(q.buffer)
}

func (q *queue) run() {
	defer func() {
		q.wg.Add(-len(q.buffer))
		q.buffer = nil
	}()

	for {
		select {
		case val := <-q.push:
			q.buffer = append(q.buffer, val)
			q.wg.Add(1)
		case <-q.pop:
			q.count--
		case suspend := <-q.suspend:
			if suspend != q.suspended {
				if suspend {
					q.wg.Add(1)
				} else {
					q.wg.Done()
				}

				q.suspended = suspend
			}
		case <-q.stop:
			q.stopped = true
		}

		if q.stopped && q.count == 0 {
			return
		}

		for (q.count < q.ConcurrencyLimit || q.ConcurrencyLimit == 0) && len(q.buffer) > 0 && !(q.suspended || q.stopped) {
			val := q.buffer[0]
			q.buffer = q.buffer[1:]
			q.count++

			go func() {
				defer func() {
					q.pop <- struct{}{}
					q.wg.Done()
				}()

				q.Handler(val)
			}()
		}
	}
}
