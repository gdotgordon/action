package action

import (
	"encoding/json"
	"testing"
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
			},
			expected: []Item{
				{Action: "jump", Avg: 150},
				{Action: "ruminate", Avg: 55},
				{Action: "run", Avg: (175.0 / 3.0)},
			},
		},
	} {

		// Input all the data.
		acc := NewAccumulator()
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
