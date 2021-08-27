package main

import (
	"fmt"
	"github.com/fobilow/detach"
	"time"
)

func main() {
	// attach -d flag to default CommandLine flag set
	cleanup := detach.Setup("d", nil)
	defer cleanup()

	// long running process
	timer := time.NewTimer(1 * time.Minute)
	fmt.Println(<-timer.C)
}
