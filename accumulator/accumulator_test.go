package accumulator

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
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
		expError string
	}{
		{
			expected: []Item{},
		},
		{
			input:    []string{`{"time":100}`},
			expError: "missing action",
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
		acc := New()
		// Sequential adds.
		for _, inp := range data.input {
			err := acc.AddAction(inp)
			if err != nil && data.expError == "" {
				t.Fatalf("%d: got unexpcted error: %v", n, err)
			}

			if data.expError != "" {
				if err == nil {
					t.Fatalf("%d: did not get expected error: %v", n, data.expError)
				} else if !strings.Contains(err.Error(), data.expError) {
					t.Fatalf("%d: got error '%s', but it did not contain '%s'",
						n, err, data.expError)
				}
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

// Test concurrent adds and stats retrievals.
func TestConcurrency(t *testing.T) {
	// The test creates a number of distinct actions, each of which call AddAction()
	// and then GetStats() the identical number of times.  Each individual
	// AddAction/GetStats pair is called concurrently in a goroutine in one
	// of the configured number of worker goroutines.
	//
	// In summary:
	// 1. The actions are named action00001, action00002, ....up to numActions,
	//     the configured number of actions.  The action0000x format is to preserve
	//     numbering order when looping thorugh the results, which are alphabetized
	//     by action name.
	// 2. Each action will do the number of iterations, specified by numIters.
	// 3. The number of worker goroutines is numwWorkers.
	// 4. Each action will add actions with times (action #)*1, (action #)*2, etc.
	// 5. Given #4, action 1 will add the times 1,2,3,4,5....numIters
	// 6. Action 2 will add the items 2,4,6,8,10 ... numIters*2
	// 7. The nth action will add the items n*1,n*2,n*3, ... n*numIters
	// 8. These actions will be submitted in completely randomized order to the
	//    workers.
	// 9. Observe that the sum of the numbers 1...n is (n*(n+1))/2.0, hence the expected
	//    average for the first action is (numIters*(numIters+1))/2.0*numIters = (numIters+1)/2.0,
	//    hence for the nth action, the expected average is:
	//        (n*(numIters)*(numIters+1))/2.0*numIters = (n*(numIters+1))/2.0
	// Example: action003, 4 iteration time values: 3, 6, 9, 12.  Average is (3*5)/2 = 7.5
	//
	const numActions = 109
	const numIters = 304
	const numWorkers = 20

	// The numbers 0...(numActions*numIters) are the number of distinct calls to AddAction().
	// Using division and modular arithmentic we can ensure each distinct call is made exactly
	// once.  Here we randomize the order of the calls for submission to the workers.
	rand.Seed(time.Now().Unix())
	perm := rand.Perm(numActions * numIters)

	// Create a new accumulator to get aand fetch the results from.
	acc := New()

	// Goroutine to feed action tasks into the workers and finally close
	// the channel.
	jobChan := make(chan int)
	go func(wrtr chan<- int) {
		for _, p := range perm {
			wrtr <- p
		}
		close(wrtr)
	}(jobChan)

	// Goroutine to process the results.  We'll accumulate any errors
	// and at the end to see if we got the expcted error-free run.
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

	// Create the worker goroutines.  Each will receive a job number through
	// it's input channel and infer the unique action and time from that.
	// It will then call AddAction() for this action, and also call GetStats().
	// Successes (nils) and errors for both calls will be sent to the error
	// receiver channel.  Note there isn't much predictable info we can gather
	// from GetStats() midstream of the test, but at least we can check that the
	// result is well-formed and that there were no race condition errors in the code.
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

	// Wait for the error groutine to exit.
	ewg.Wait()

	// We should have received one result for each call to add and each stats call.
	if resCnt != 2*numActions*numIters {
		t.Fatalf("expected %d actions, but got %d",
			2*numActions*numIters, resCnt)
	}

	// We don't expect any errors to have occurred.
	for _, e := range errs {
		if e != nil {
			t.Fatalf("got unexpcted error: %v", e)
		}
	}

	// Get the final stats and make sure they are all present and correct.
	var items []Item
	stats := acc.GetStats()
	if err := json.Unmarshal([]byte(stats), &items); err != nil {
		t.Fatalf("error unmarshaling final stats: %v", err)
	}
	if len(items) != numActions {
		t.Fatalf("expected %d acitons, got %d", numActions, len(items))
	}

	// To see result, comment this code out.
	//fmt.Println(acc.GetStats())

	// Check the veracity of each item.
	for i, item := range items {
		actionNum := i + 1
		aname := fmt.Sprintf("action%05d", actionNum)
		if item.Action != aname {
			t.Fatalf("expected action '%s', but got '%s", aname, item.Action)
		}

		// The algorithm for calcualting the expceted average was detailed at the
		// top of the function.
		expAvg := float64(actionNum*(numIters+1.0)) / 2.0
		if item.Avg != float64(expAvg) {
			t.Fatalf("%d: expected average '%f', but got '%f", i, expAvg, item.Avg)
		}
	}
}
