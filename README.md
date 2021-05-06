# action
Calculates the average times for the user's action data.

## Requirements
* Go toolchain, preferably Go 16.x, any platform that supports Go

## Introduction and Overview
This code implements the requred API, including two functions
1. add an action item (via a JSON string) that speciies and action name and a time value of some arbitrary unit.
2. return the average time value for all actions added as a string representing a JSON array

## Implementation
The solution uses a hash map along with a sorted list of keys to maintain the per-action data.  Note, as the objective is to only aggrgate data, and not examine individual actions, it is sufficient to keep only cumulative information for each action, that is the count and average.  The significantly reduces the complexity of managing the data and calculating statisitics.  This design is flexible enough so that we could aggregate other statitics such as min and max values, most common value, etc.

Concurrency is achieved via a single writer, mutiple reader mutex.  That is a single writer can run when there are no readers, and any number of readers can run when there are no writers.  This mechanism is a good choice to maximize throughput for this kind of paradigm.

In addition, each action entry is stored in a format that can be serialized directly to JSON to reduce the amount of object creation.  In fact the hash map stores a pointer to each action item, so that the pointers can be quickly copied out to an array to pass to the Go JSON serializer.

### Test Cases
The are two unit tests and one integration-like conccurrency test in `accumulator/accumulator_test.go`.  In production, we could separate out the integration tests from the unit test and require a special test arg to run it separately, but for the purposes of this exercise, the three tests are grouped together for convenience of evaluation.

