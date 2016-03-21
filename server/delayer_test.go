package server

import (
	"sync"
	"time"

	. "gopkg.in/check.v1"
)

type TimerSuite struct{}

var _ = Suite(&TimerSuite{})

type SimpleFnHolder struct {
	sync.Mutex
	Calls []string
}

func (s *SimpleFnHolder) CallFn(key string) func() {
	return func() {
		s.Lock()
		defer s.Unlock()

		s.Calls = append(s.Calls, key)
	}
}

func (s *SimpleFnHolder) AssertCalls(c *C, key ...string) {
	s.Lock()
	defer s.Unlock()

	c.Assert(s.Calls, HasLen, len(key))
	for i := range s.Calls {
		c.Assert(s.Calls[i], Equals, key[i])
	}
}

func (t *TimerSuite) TestSingleDelay(c *C) {
	s := SimpleFnHolder{}
	f := NewFunctionDelayer(100 * time.Millisecond)
	f.Delay("abc", s.CallFn("abc"))
	time.Sleep(75 * time.Millisecond)
	// After 75ms, it shouldn't have been called yet
	s.AssertCalls(c)
	time.Sleep(75 * time.Millisecond)
	// After 150 ms, it should have been called
	s.AssertCalls(c, "abc")
	time.Sleep(100 * time.Millisecond)
	// After another 100 ms, it should still only have one call
	s.AssertCalls(c, "abc")
}

func (t *TimerSuite) TestMultipleDelays(c *C) {
	s := SimpleFnHolder{}
	f := NewFunctionDelayer(100 * time.Millisecond)
	f.Delay("abc", s.CallFn("abc"))
	time.Sleep(75 * time.Millisecond)
	// After 75ms, it shouldn't have been called yet
	s.AssertCalls(c)
	// Reset the timer by calling it again
	f.Delay("abc", s.CallFn("abc"))
	time.Sleep(75 * time.Millisecond)
	s.AssertCalls(c)
	time.Sleep(75 * time.Millisecond)
	// After 225 ms total, it should have been called
	s.AssertCalls(c, "abc")
}

func (t *TimerSuite) TestLateDelay(c *C) {
	s := SimpleFnHolder{}
	f := NewFunctionDelayer(100 * time.Millisecond)
	f.Delay("abc", s.CallFn("abc"))
	time.Sleep(75 * time.Millisecond)
	// After 75ms, it shouldn't have been called yet
	s.AssertCalls(c)
	time.Sleep(75 * time.Millisecond)
	// After 150ms, it should have been called
	s.AssertCalls(c, "abc")
	// Delaying the same key should trigger a new call
	f.Delay("abc", s.CallFn("abc"))
	time.Sleep(125 * time.Millisecond)
	// After 125 ms more, it should have been called again
	s.AssertCalls(c, "abc", "abc")
}

func (t *TimerSuite) TestMultipleKeysAndDelays(c *C) {
	s := SimpleFnHolder{}
	f := NewFunctionDelayer(100 * time.Millisecond)
	f.Delay("abc", s.CallFn("abc")) // triggers at 100ms
	f.Delay("def", s.CallFn("def")) // triggers at 100ms
	time.Sleep(75 * time.Millisecond)
	// After 75ms, nothing should have been called yet
	s.AssertCalls(c)
	// Reset the timer for "abc" by calling it again
	f.Delay("abc", s.CallFn("abc")) // now triggers at 175ms
	time.Sleep(75 * time.Millisecond)
	// After 150 ms total, only "def" should have been called
	s.AssertCalls(c, "def")
	// Delay a new key: "ghi"
	f.Delay("ghi", s.CallFn("ghi")) // triggers at 250ms
	time.Sleep(75 * time.Millisecond)
	// After 225 ms total, "def" and "abc" should have been called
	s.AssertCalls(c, "def", "abc")
	time.Sleep(75 * time.Millisecond)
	// After 300ms total, "ghi" should be called too
	s.AssertCalls(c, "def", "abc", "ghi")
}
