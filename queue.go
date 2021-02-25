package log

import (
	"runtime"
	"sync"
)

// queue is a data structure that executes jobs asynchronously in a goroutine.
//
// The operations of the queue (pushJob, stopQueue, ...) are not thread-safe.
// If the queue is to be executed in different goroutines, suitable measures must still be added.
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

// Handler is the function that executes individual jobs.
// The parameter of the handler function is the object that is added to the queue.
type handler func(interface{})

// NewQueue creates a new queue.
// As parameter the handler function and the maximum number of parallel jobs needed.
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

// pushJob adds a new job to the queue.
func (q *queue) pushJob(val interface{}) {
	q.push <- val
}

// stopQueue stops the queue. After that no new job can be added.
// Jobs that still exist on the queue are still processed in the background.
func (q *queue) stopQueue() {
	q.stop <- struct{}{}
	runtime.SetFinalizer(q, nil)
}

// wait waits until there are no more jobs in the queue.
func (q *queue) wait() {
	q.wg.Wait()
}

// len returns the number of jobs in the queue.
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
