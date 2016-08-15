package main

type task interface {
	Process()
}

type queue struct {
	tasks chan task
}

func newQueue(capacity int) *queue {
	pool := &queue{
		tasks: make(chan task),
	}

	for i := 1; i <= capacity; i++ {
		go func() {
			for {
				(<-pool.tasks).Process()
			}
		}()
	}

	infof("%d task threads has been allocated", capacity)

	return pool
}

func (pool *queue) push(task task) {
	pool.tasks <- task
}
