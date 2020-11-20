package version

import (
	"encoding/json"
	"fmt"
)

var (
	// BuildDate is the date when the binary was built
	BuildDate string
	// GitCommit is the commit hash when the binary was built
	GitCommit string
	// BinaryVersion is the version of the KMS binary
	BuildVersion string
	APIVersion   = "v1beta1"
	Runtime      = "Microsoft AzureKMS"
)

// PrintVersion prints the current KMS plugin version
func PrintVersion() (err error) {
	pv := struct {
		BuildVersion string
		GitCommit    string
		BuildDate    string
	}{
		BuildDate:    BuildDate,
		BuildVersion: BuildVersion,
		GitCommit:    GitCommit,
	}

	var res []byte
	if res, err = json.Marshal(pv); err != nil {
		return
	}

	fmt.Printf(string(res) + "\n")
	return
}
