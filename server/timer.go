package server

import (
	"sync"
	"time"
)

// FunctionDelayer manages functions, identified by keys, such that they are called after a set period of time.
// If a request to schedule a function call uses a key for a function that has already been scheduled, it resets
// the delay for that call.  For example, consider a FunctionDelayer f with a configured delay of 3 seconds.
// If f.Delay("foo", myFunc) is invoked only once, myFunc() will be called 3 seconds later.  If
// f.Delay("bar", myFunc) is invoked once, and then invoked again 2 seconds later, myFunc() will be called 3
// seconds after the second f.Delay invocation (for a total of 5 seconds after the first f.Delay invocation).
type FunctionDelayer struct {
	sync.Mutex
	Duration time.Duration
	timers   map[string]*time.Timer
}

// NewFunctionDelayer creates a new FunctionDelayer with the specified duration.
func NewFunctionDelayer(duration time.Duration) *FunctionDelayer {
	f := new(FunctionDelayer)
	f.Duration = duration
	f.timers = make(map[string]*time.Timer)
	return f
}

// Delay schedules a function to be called after FunctionDelayer's Duration has elapsed.  Subsequent calls to
// Delay with the same key will reset the duration timer if it is still active.  Note that on these subsequent
// calls, the passed in function is ignored (and only the original function is invoked when the timer expires).
func (f *FunctionDelayer) Delay(key string, fn func()) {
	f.Lock()
	defer f.Unlock()

	if t, ok := f.timers[key]; ok {
		// timer exists, so reset it!
		if ok := t.Reset(f.Duration); !ok {
			// timer expired, so add a new one
			f.addNewTimer(key, fn)
		}
	} else {
		// no timer... add one!
		f.addNewTimer(key, fn)
	}
}

func (f *FunctionDelayer) addNewTimer(key string, fn func()) {
	f.timers[key] = time.AfterFunc(f.Duration, func() {
		f.Lock()
		delete(f.timers, key)
		f.Unlock()
		fn()
	})
}
