// This is a trivial sample app showing usage of the package and API.
package main

import (
	"fmt"
	"os"

	"github.com/gdotgordon/action/accumulator"
)

func main() {

	// Create a new accumulator.
	acc := accumulator.New()

	// Call addAction() for some actions.
	for _, inp := range []string{
		`{"action":"jump", "time":100}`,
		`{"action":"run", "time":75}`,
		`{"action":"jump", "time":200}`,
	} {
		if err := acc.AddAction(inp); err != nil {
			fmt.Println("AddAction error:", err)
			os.Exit(1)
		}
	}

	// Get and print the statistics result.
	fmt.Println(acc.GetStats())
}
