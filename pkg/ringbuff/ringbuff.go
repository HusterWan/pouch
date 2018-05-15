package ringbuff

import (
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

type ringNode struct {
	next  *ringNode
	value interface{}
	no    int
}

// RingBuff implements a circular list.
type RingBuff struct {
	sync.Mutex
	cond    *sync.Cond
	pushPtr *ringNode
	popPtr  *ringNode
	closed  bool
}

// New creates a RingBuff.
func New(n int) *RingBuff {
	var (
		first *ringNode
		last  *ringNode
	)

	for i := 0; i < n; i++ {
		p := &ringNode{no: i}

		if last != nil {
			last.next = p
		} else {
			first = p
		}

		last = p
	}

	last.next = first

	r := &RingBuff{
		pushPtr: first,
		popPtr:  first,
		cond:    sync.NewCond(&sync.Mutex{}),
	}
	logrus.Debug("%p init stat %d %d", r, first.no, first.no)

	return r
}

// Push puts a elemnet into RingBuff and returns the status of covering or not.
func (r *RingBuff) Push(value interface{}) bool {
	r.Lock()
	defer r.Unlock()

	cover := false

	if r.closed {
		return cover
	}

	if r.pushPtr.value != nil {
		cover = true
	}

	// store value
	r.pushPtr.value = value

	// wakes the "Pop" goroutine waiting on "cond".
	r.cond.Broadcast()

	// move pointer to next node.
	r.pushPtr = r.pushPtr.next
	logrus.Debug("%p after push %d %d", r, r.pushPtr.no, r.popPtr.no)

	return cover
}

// Pop returns a element, if RingBuff is empty, Pop() will block.
func (r *RingBuff) Pop() (interface{}, bool) {
	r.Lock()

	if v := r.popPtr.value; v != nil {
		isClosed := r.closed

		r.popPtr.value = nil // if we readed the node, must set nil to it.
		// move to next node.
		r.popPtr = r.popPtr.next
		logrus.Debug("%p after pop has data %d %d %v", r, r.pushPtr.no, r.popPtr.no, v)

		// NOTICE: unlock
		r.Unlock()

		return v, isClosed
	}

	if r.closed {
		isClosed := r.closed

		// NOTICE: unlock
		r.Unlock()

		return nil, isClosed
	}

	// block util there is one element at least.
	r.cond.L.Lock()
	for r.popPtr.value == nil && !r.closed {
		// NOTICE: unlock, then to wait. if not call "Unlock", will block other's operation, eg: Push().
		r.Unlock()

		r.cond.Wait()

		// NOTICE: Wait() return, need to hold lock again.
		r.Lock()
	}

	v := r.popPtr.value
	isClosed := r.closed
	r.popPtr.value = nil // if we readed the node,  must set nil to it.
	// move to next node.
	r.popPtr = r.popPtr.next
	logrus.Debug("%p after pop end wait %d %d %v", r, r.pushPtr.no, r.popPtr.no, v)

	r.cond.L.Unlock()

	r.Unlock()

	return v, isClosed
}

// Close closes the RingBuff.
func (r *RingBuff) Close() error {
	// first try to wakeup
	r.Lock()
	if r.closed {
		r.Unlock()
		return nil
	}
	r.cond.Broadcast()
	r.Unlock()

	for {
		r.Lock()
		if r.pushPtr == r.popPtr {
			// unlock
			r.Unlock()
			break
		}

		// unlock
		r.Unlock()
		time.Sleep(time.Millisecond * 10)
	}

	r.Lock()
	r.closed = true
	r.cond.Broadcast()
	r.Unlock()

	return nil
}
