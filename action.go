package action

import (
	"encoding/json"
	"strings"
	"sync"
)

// Input represents the unmarshalled input of AddAction.
type Input struct {
	Action string  `json:"action"`
	Time   float64 `json:"time"`
}

// Item holds the data for a given action item.  It is intended to be
// serializable directly as JSON for the statistics query.  Note the
// count field, used internally, is not part of the serialized JSON.
type Item struct {
	Action string  `json:"action"`
	Avg    float64 `json:"avg"`
	count  int
}

// Accumulator is the holder of all the actions and their various metrics.
// Note, there is no need to track every single action item submitted.
// Instead, we'll maintain a running average plus the item count, so we
// can simply adjust the average as a new item comes in.
type Accumulator struct {

	// Hash map of action key to action item.
	items map[string]*Item

	// Sorted list of hash map keys.  It is faster to maintain a sorted
	// list to insert into, rather than to build an entire sorted list of
	// keys each time a new key is added.
	keys []string

	// We can have multiple readers querying the data for stats, but only
	// an exclusive writer with no readers may update.
	mu sync.RWMutex
}

// NewAccumulator returns the accumulator, which implements the required API.
func NewAccumulator() *Accumulator {
	acc := Accumulator{
		items: make(map[string]*Item),
	}
	return &acc
}

// AddAction adds a list of actions and times, recalcuates the appropriate
// running averages, and adds to the sorted list if needed.  We need
// exclusive write access for this.
func (acc *Accumulator) AddAction(actions string) error {
	// First make sure we have valid input before grabbing the lock.
	var input *Input
	if err := json.Unmarshal([]byte(actions), &input); err != nil {
		return err
	}

	acc.mu.Lock()
	defer acc.mu.Unlock()
	item, ok := acc.items[input.Action]
	if !ok {
		// Add a new action key to the sorted list and item map.
		acc.keys = insertSorted(input.Action, acc.keys)
		acc.items[input.Action] = &Item{Action: input.Action, Avg: input.Time, count: 1}
	} else {
		// Recalculate the running average for this action item and bump the count.
		item.Avg = (float64(item.count)*item.Avg + input.Time) / float64(item.count+1)
		item.count++
	}
	return nil
}

// GetStats returns all the data for all actions.
func (acc *Accumulator) GetStats() string {

	// We've stord the data in the proper format, so we simply need to
	// copy the pointers from the hash map, ordered by sorted keys, to an
	// array to marshall for the desired string result.
	acc.mu.RLock()
	res := make([]*Item, len(acc.keys))
	for i, k := range acc.keys {
		res[i] = acc.items[k]
	}
	acc.mu.RUnlock()

	b, err := json.MarshalIndent(res, "", "  ")
	if err != nil {
		// If the marshall fails, something is fundamentally broken
		// in the system.
		panic("cannot marshall internal data: " + err.Error())
	}
	return string(b)
}

// insertSorted inserts a key in its proper place in a sorted list.
// If the item is found, it simply returns the list unchanged.
func insertSorted(key string, list []string) []string {

	// The algorithm is to use a binary search to find the first key greater
	// than or equal to the input key.  This will have O(log n) execution time.
	low := 0
	high := len(list) - 1
	for low <= high {
		mid := (low + high) / 2
		switch strings.Compare(key, list[mid]) {
		case 0:
			return list
		case -1:
			high = mid - 1
		case 1:
			low = mid + 1
		}
	}

	// low will now be at the desired offset because after narrowing in,
	// low was at minimum greater than they key, and high is now one
	// less than low, so the loop exited.
	if low == len(list) {
		// Bigger than everything in list, so append.
		return append(list, key)
	}

	// Insert the item before "low", which is now at the right spot.
	return append(list[0:low], append([]string{key}, list[low:]...)...)
}
