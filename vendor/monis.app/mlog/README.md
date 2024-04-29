`mlog` is an opinionated structured log package for Kubernetes devs who are stuck using `klog`.

[![Go Reference](https://pkg.go.dev/badge/monis.app/mlog.svg)](https://pkg.go.dev/monis.app/mlog)

```go
package main

import "monis.app/mlog"

func main() {
	if err := mainErr(); err != nil {
		mlog.Fatal(err)
	}
}

func mainErr() error { // return an error instead of mlog.Fatal to allow defer statements to run
	defer mlog.Setup()()  // set up log flushing and attempt to flush on exit

	// use Always for logs emitted before config parsing
	mlog.Always("Running server", "version", version)

	// set up ctx and logConfig

	if err := mlog.ValidateAndSetLogLevelAndFormatGlobally(ctx, logConfig); err != nil {
		return fmt.Errorf("validate log level: %w", err)
	}

	// run server
}
```
