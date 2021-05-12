package log

import (
	"sync"
)

// queue is a data structure that executes jobs asynchronously in a goroutine.
//
// The operations of the queue (addJob, close, ...) are not thread-safe.
// If the queue is to be executed in different goroutines, suitable measures must still be added.
type queue struct {
	jobs    chan interface{}
	closeq  chan bool
	workers chan chan bool
	once    sync.Once
	handler handler
}

// Handler is the function that executes individual jobs.
// The parameter of the handler function is the object that is added to the queue.
type handler func(interface{})

// NewQueue creates a new queue.
// As parameter the handler function, the maximum number of parallel jobs and the size is needed.
func newQueue(handler func(interface{}), workers int, size int) *queue {
	q := &queue{
		jobs:    make(chan interface{}, size),
		closeq:  make(chan bool),
		workers: make(chan chan bool, workers),
		handler: handler,
	}

	for w := 0; w < workers; w++ {
		q.workers <- q.worker()
	}

	close(q.workers)
	return q
}

// worker creates a worker that runs tasks in the background.
func (q *queue) worker() chan bool {
	done := make(chan bool)

	go func() {
	work:
		for {
			select {
			case <-q.closeq:
				break work
			case j := <-q.jobs:
				q.handler(j)
			}
		}

		close(done)
	}()

	return done
}

// close closes the queue. After that no new job can be added.
// Jobs that still exist on the queue are still processed and the function blocks until all tasks are completed.
func (q *queue) close() {
	q.once.Do(func() {
		close(q.closeq)
	})

	for w := range q.workers {
		<-w
	}

	close(q.jobs)
	for j := range q.jobs {
		q.handler(j)
	}
}

// addJob adds a new job to the queue.
func (q *queue) addJob(job interface{}) bool {
	// Check the queue is open first
	select {
	case <-q.closeq:
		return false
	default:
		// While the jobs queue send is blocking, we might shutdown the queue
		select {
		case q.jobs <- job:
			return true
		case <-q.closeq:
			return false
		}
	}
}
