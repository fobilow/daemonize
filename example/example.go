package main

import (
	"fmt"
	"time"

	"github.com/fobilow/detach"
)

func main() {
	// attach -d flag to default CommandLine flag set
	cleanup := detach.Setup("d", nil)
	defer cleanup()

	// long running process
	timer := time.NewTimer(1 * time.Minute)
	fmt.Println(<-timer.C)
}
