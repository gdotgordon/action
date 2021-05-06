package action

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"
)

// Testing algorithm to insert key into sorted list.
func TestInsertSorted(t *testing.T) {
	for n, data := range []struct {
		key      string
		list     []string
		expected []string
	}{
		{
			key:      "a",
			list:     nil,
			expected: []string{"a"},
		},
		{
			key:      "b",
			list:     []string{"a", "c", "d"},
			expected: []string{"a", "b", "c", "d"},
		},
		{
			key:      "c",
			list:     []string{"a", "b", "d", "e"},
			expected: []string{"a", "b", "c", "d", "e"},
		},
		{
			key:      "d",
			list:     []string{"a", "b", "c"},
			expected: []string{"a", "b", "c", "d"},
		},
		{
			key:      "a",
			list:     []string{"b", "c", "d"},
			expected: []string{"a", "b", "c", "d"},
		},
		{
			key:      "a",
			list:     []string{"a", "b", "c", "d"},
			expected: []string{"a", "b", "c", "d"},
		},
	} {
		res := insertSorted(data.key, data.list)
		if len(res) != len(data.expected) {
			t.Fatalf("%d: result length %d does not match expected length %d",
				n, len(res), len(data.expected))
		}
		for i, v := range data.expected {
			if v != res[i] {
				t.Fatalf("%d: list item does not match expected: %s, %s", n, v, res[i])
			}
		}
	}
}

// Testing basic functionality without concurrencey.
func TestBasicOperation(t *testing.T) {
	for n, data := range []struct {
		input    []string
		expected []Item
		expError error
	}{
		{
			expected: []Item{},
		},
		{
			input:    []string{`{"action":"jump", "time":100}`},
			expected: []Item{{Action: "jump", Avg: 100}},
		},
		{
			input: []string{
				`{"action":"jump", "time":100}`,
				`{"action":"jump", "time":200}`,
			},
			expected: []Item{
				{Action: "jump", Avg: 150},
			},
		},
		{
			input: []string{
				`{"action":"jump", "time":100}`,
				`{"action":"run", "time":75}`,
				`{"action":"jump", "time":200}`,
			},
			expected: []Item{
				{Action: "jump", Avg: 150},
				{Action: "run", Avg: 75},
			},
		},
		{
			input: []string{
				`{"action":"run", "time":75}`,
				`{"action":"jump", "time":100}`,
				`{"action":"run", "time":25}`,
				`{"action":"jump", "time":200}`,
			},
			expected: []Item{
				{Action: "jump", Avg: 150},
				{Action: "run", Avg: 50},
			},
		},
		{
			input: []string{
				`{"action":"run", "time":75}`,
				`{"action":"run", "time":75}`,
				`{"action":"jump", "time":100}`,
				`{"action":"run", "time":25}`,
				`{"action":"ruminate", "time":100}`,
				`{"action":"ruminate", "time":10}`,
				`{"action":"jump", "time":200}`,
				`{"action":"think", "time":0}`,
			},
			expected: []Item{
				{Action: "jump", Avg: 150},
				{Action: "ruminate", Avg: 55},
				{Action: "run", Avg: (175.0 / 3.0)},
				{Action: "think", Avg: 0},
			},
		},
	} {
		// Input all the data.
		acc := NewAccumulator()
		// Sequential adds.
		for _, inp := range data.input {
			err := acc.AddAction(inp)
			if err != nil && data.expError == nil {
				t.Fatalf("%d: got unexpcted error: %v", n, err)
			}
			if err == nil && data.expError != nil {
				t.Fatalf("%d: did not get unexpcted error: %v", n, data.expError)
			}
		}

		// Comapre the results with what's expected.
		res := acc.GetStats()
		var items []Item
		if err := json.Unmarshal([]byte(res), &items); err != nil {
			t.Fatalf("%d: error unmarshalling result: %v", n, err)
		}

		if len(items) != len(data.expected) {
			t.Fatalf("%d: result length %d does not match expected length %d",
				n, len(items), len(data.expected))
		}
		for i, v := range data.expected {
			if v != items[i] {
				t.Fatalf("%d: list item does not match expected: %v, %v", n, items[i], v)
			}
		}
	}
}

func TestConcurrency(t *testing.T) {
	const numActions = 100
	const numIters = 100
	const numWorkers = 10

	rand.Seed(time.Now().Unix())
	perm := rand.Perm(numActions * numIters)

	acc := NewAccumulator()

	// Goroutine to feed action tasks into the workers.
	jobChan := make(chan int)

	go func(wrtr chan<- int) {
		for _, p := range perm {
			wrtr <- p
		}
		close(wrtr)
	}(jobChan)

	// Goroutine to process the results.  We'll accumulate any errors
	// and at the end see if we got the expcted error-free run.
	resChan := make(chan error)
	var errs []error
	resCnt := 0
	var ewg sync.WaitGroup
	ewg.Add(1)
	go func(err <-chan error) {
		defer ewg.Done()
		for e := range err {
			resCnt++
			if e != nil {
				errs = append(errs, e)
			}
		}
	}(resChan)

	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {

		wg.Add(1)
		go func(job <-chan int, res chan<- error) {
			defer wg.Done()

			for j := range job {
				action := (j / numIters) + 1
				time := (j % numIters) + 1
				req := fmt.Sprintf(`{"action": "action%05d", "time": %d}`, action, action*time)
				res <- acc.AddAction(req)
				stats := acc.GetStats()
				var items []Item
				res <- json.Unmarshal([]byte(stats), &items)
			}
		}(jobChan, resChan)
	}

	// Wait for all the jobs to be processed and close the error
	// channel to terminate the error processing goroutine.
	wg.Wait()
	close(resChan)
	ewg.Wait()

	if resCnt != 2*numActions*numIters {
		t.Fatalf("expected %d actions, but got %d",
			2*numActions*numIters, resCnt)
	}

	for _, e := range errs {
		if e != nil {
			t.Fatalf("got unexpcted error: %v", e)
		}
	}

	var items []Item
	stats := acc.GetStats()
	if err := json.Unmarshal([]byte(stats), &items); err != nil {
		t.Fatalf("error unmarshaling final stats: %v", err)
	}
	if len(items) != numActions {
		t.Fatalf("expected %d acitons, got %d", numActions, len(items))
	}

	fmt.Println(acc.GetStats())
	for i, item := range items {
		actionNum := i + 1
		aname := fmt.Sprintf("action%05d", actionNum)
		if item.Action != aname {
			t.Fatalf("expected action '%s', but got '%s", aname, item.Action)
		}
		expAvg := float64((actionNum)) * numIters * (numIters + 1.0) / (2.0 * numIters)
		fmt.Println("exp avg")
		if item.Avg != float64(expAvg) {
			t.Fatalf("%d: expected average '%f', but got '%f", i, expAvg, item.Avg)
		}
	}
}
