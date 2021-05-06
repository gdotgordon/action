package main

import (
	"fmt"
	"os"

	"github.com/gdotgordon/action/accumulator"
)

func main() {
	acc := accumulator.New()

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
	fmt.Println(acc.GetStats())
}
