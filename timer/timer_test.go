package timer

import (
	"fmt"
	"time"
)

// Example of combining a timer with a defer to make it easy to put all your
// timing code at the top of a function.
func Example() {
	defer func(t Timer) {
		dur := t.Finish()
		fmt.Printf("log my duration as %g\n", dur)
	}(Start())
}

// Example_block when timing non-funciton based locks, separate the start and finish
func Example_block() {
	// do some work
	t := Start()
	// do some more work
	dur := t.Finish()
	fmt.Printf("log my duration as %g\n", dur)
}

// Example_othertime for when starting from Now isn't quite right
func Example_otherTime() {
	actualStart := time.Unix(1525150486, 0)
	t := New(actualStart)
	// do some work
	dur := t.Finish()
	fmt.Printf("log my duration as %g\n", dur)
}
