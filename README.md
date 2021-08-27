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
## Available Options
- `app -d start` - start app in detach mode
- `app -d stop` - stop apps running in detach mode
- `app -d restart` - restart app running in detach mode
- `app -d status` - view status of apps running in detach mode
    