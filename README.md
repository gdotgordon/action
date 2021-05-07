# action
Calculates the average times for the user's action data.

## Requirements
* Go toolchain, preferably Go 16.x, any platform that supports Go

## Introduction and Overview
This code implements the required API, including two functions:
1. add an action item (via a JSON string) that specifies and action name and a time value of some arbitrary unit
2. return the average time value for all actions added as a string representing a JSON array

## Run Instructions
* clone this repo, and from the top-level folder ("action") from the shell of your choice, you can run the sample app and test cases:
  1. `go run .` runs the sample app
  2. `go test -v -count=1 ./...` runs the test cases (the "count" flag is to force clearing any cache results in case you run multiple times).

## Source File Locations
* `main.go` in the top level folder is the aforementioned sample app that shows the usage of the API.
* The `accumulator` folder contains the implementation of the API in `accumulator.go`.  The test cases are all in accumulator_test.go`.
* Note: in a typical Go utility package, the top level would contain the root of the implementation, but I moved it down one level so I could include a main program in the top-level directory. 

## API summary:
* To use the accumulator package, you can import it: `import "github.com/gdotgordon/action/accumulator"`
* To instantiate the main accumulator type, you'd use something like: `acc := accumulator.New()`
* To add an action, use `err := AddAction(action string)`, where `err` is any error returned for invalid user input, and `action` is the JSON for an action to add, for example:
```
`{"action":"jump", "time":100}`
```
* To get a summary of all the actions and their averages use `resJSON := acc.GetStats()`.  This function returns a string containing the JSON array of actions and the averages, sorted alphabetically by action name.  The sorting arguably gives the most useful and polished result.  Here's a sample output:

```
$ go run .
[
  {
    "action": "jump",
    "avg": 150
  },
  {
    "action": "run",
    "avg": 75
  }
]
```

## Implementation Notes
The solution uses a hash map along with a list of sorted keys to maintain the per-action data.  Note, as the objective is only to aggregate data, and not examine individual actions, it is sufficient to keep only cumulative information for each action, that is the count and average.  The significantly reduces the performance and complexity of managing the data and calculating statistics.  This design is flexible enough so that we could easily aggregate other statistics such as min and max values, most common value, etc.

As it turns out, due to tiny roundoff errors in remote decimal places that come from maintaining a running average, I decided to also store and use the running sum to get the average, so compares in my test cases would not tell me that .05 was not equal to 0.5.  In production, we could implement this either way, depending on the precision requirements.

Since we need to return the actions in sorted order and Go doesn't have a built-in ordered hash map, we maintain a sorted list of all the keys entered and update the list in real time as new actions are added.  I wrote a simple binary search/insert algorithm which is O(log n) to do the inserts if the action added has not yet been seen.  The other option of completely sorting the list for each stats request would use something like the built-in O(n * log n) sort.  If the list is maintained in sorted order, then adding a new key by say appending it to the list and then sorting that via an O(n * log n) would be a waste of CPU cycles, given the list is already nearly sorted.

Concurrency is achieved via a single writer, multiple reader mutex.  That is a single writer can run when there are no readers, and any number of readers can run when there are no writers.  This mechanism is a good choice to maximize throughput for this kind of paradigm.  The code also attempts to minimize the critical sections, for example, by doing JSON marshaling outside those critical regions.

In addition, each action entry is stored in the hash map a format that can be serialized directly to JSON to reduce the amount of object creation.  In fact, the hash map stores a pointer to each action item, so that the pointers can be quickly copied out to an array to pass to the Go JSON serializer.

## Test Cases
There are two unit tests and one integration-like concurrency test in `accumulator/accumulator_test.go`.  We could easily separate out the integration tests from the unit test and require a special test arg to run it separately, but for the purposes of this exercise, the three tests are grouped together for convenience of perusal.  The unit tests use the Go-recommended table-driven approach, where new cases are added by adding them to an array of per-case test data.

The concurrency test is essentially an integration test.  It interleaves add action and statistics requests in multiple goroutines.  Importantly, this test has a predictable set of results, so we can feel confident about the correctness of the code.  The test helped identify, locate and correct a race condition.

## Conclusion
I've strived to make this package polished and well-documented.  As with any Go code, I enjoyed coming up with and implementing the solution.

