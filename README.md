# Detach
Add detach mode to any go CLI application

## Example
```go
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
	fmt.Println(<- timer.C)
}

```
Then you can run the app in daemon mode by adding the flag:
`app -d`
    