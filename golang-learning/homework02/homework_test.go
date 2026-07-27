package homework02

import (
	"math"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// Q1
func TestAddTen(t *testing.T) {
	val := 5
	AddTen(&val)
	if val != 15 {
		t.Errorf("Expected 15, got %d. Did you implement AddTen?", val)
	}
}

// Q2
func TestDoubleSlice(t *testing.T) {
	slice := []int{1, 2, 3}
	DoubleSlice(&slice)
	expected := []int{2, 4, 6}
	for i, v := range slice {
		if v != expected[i] {
			t.Errorf("Index %d: Expected %d, got %d. Did you implement DoubleSlice?", i, expected[i], v)
		}
	}
}

// Q3
func TestPrintOddEven(t *testing.T) {
	done := make(chan bool)
	go func() {
		PrintOddEven()
		done <- true
	}()

	select {
	case <-done:
		// Success
	case <-time.After(2 * time.Second):
		t.Error("Timeout: PrintOddEven took too long. Did you use WaitGroup?")
	}
}

// Q4
func TestTaskScheduler(t *testing.T) {
	var counter int32
	tasks := []func(){
		func() {
			time.Sleep(10 * time.Millisecond)
			atomic.AddInt32(&counter, 1)
		},
		func() {
			time.Sleep(10 * time.Millisecond)
			atomic.AddInt32(&counter, 1)
		},
		func() {
			time.Sleep(10 * time.Millisecond)
			atomic.AddInt32(&counter, 1)
		},
	}

	start := time.Now()
	TaskScheduler(tasks)
	duration := time.Since(start)

	// Check if all tasks executed
	if atomic.LoadInt32(&counter) != 3 {
		t.Errorf("Expected 3 tasks executed, got %d. Did you implement TaskScheduler?", counter)
	}

	// Simple check for concurrency: if sequential, it would take ~30ms. If concurrent, ~10ms + overhead.
	// We allow some buffer, but if it's strictly sequential it might be slower.
	t.Logf("Tasks took %v", duration)
}

// Q5
func TestShape(t *testing.T) {
	// Check Rectangle implementation
	r := Rectangle{Width: 10, Height: 5}
	var s Shape

	// This type assertion checks if Rectangle implements Shape
	// Use a trick to check implementation without compilation error in test if not implemented
	// But in Go static typing makes this hard. We will try to assign and handle panic if possible or just let it fail compile if user hasn't implemented methods.
	// However, since the methods are missing in the template, this file WON'T COMPILE until students add the methods.
	// To make the template compilable initially, we might need dummy methods in homework.go or comment this out.
	// BUT, the goal is TDD-like. The user's code needs to compile to run tests.
	// We'll assume the student adds the methods.

	// Check if methods exist (via interface assignment)
	s = &r
	if s.Area() != 50 {
		t.Errorf("Rectangle Area expected 50, got %f", s.Area())
	}
	if s.Perimeter() != 30 {
		t.Errorf("Rectangle Perimeter expected 30, got %f", s.Perimeter())
	}

	c := &Circle{Radius: 10}
	s = c
	expectedArea := math.Pi * 100
	if math.Abs(s.Area()-expectedArea) > 0.001 {
		t.Errorf("Circle Area incorrect")
	}
}

// Q6
func TestEmployee(t *testing.T) {
	e := Employee{
		Person:     Person{Name: "Alice", Age: 30},
		EmployeeID: "E001",
	}

	info := e.PrintInfo()
	if !strings.Contains(info, "Alice") || !strings.Contains(info, "E001") {
		t.Errorf("PrintInfo should contain Name and EmployeeID, got: %s", info)
	}
}

// Q7
func TestProducerConsumer(t *testing.T) {
	res := ProducerConsumer()
	// Only test if result is returned, otherwise assume they printed to stdout (hard to test automatically without return)
	if res != nil {
		if len(res) != 10 {
			t.Errorf("Expected 10 items, got %d", len(res))
		}
		// Check order or values
		foundOne := false
		for _, v := range res {
			if v == 1 {
				foundOne = true
			}
		}
		if !foundOne {
			t.Error("Expected to find 1 in results")
		}
	}
}

// Q8
func TestBufferedChannel(t *testing.T) {
	done := make(chan bool)
	go func() {
		BufferedChannel()
		done <- true
	}()

	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Error("BufferedChannel timed out")
	}
}

// Q9
func TestMutexCounter(t *testing.T) {
	count := MutexCounter()
	if count != 10000 {
		t.Errorf("Expected 10000, got %d. Race condition likely if not 10000.", count)
	}
}

// Q10
func TestAtomicCounter(t *testing.T) {
	count := AtomicCounter()
	if count != 10000 {
		t.Errorf("Expected 10000, got %d. Race condition likely if not 10000.", count)
	}
}
