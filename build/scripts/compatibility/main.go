// Command compatibility exports the app-owned catalog for the public website.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/tnware/freehand-stt/internal/compatibility"
	"os"
	"path/filepath"
)

func main() {
	output := flag.String("out", "site/src/data/compatibility.generated.json", "catalog destination")
	flag.Parse()
	data, err := json.MarshalIndent(compatibility.Profiles(), "", "  ")
	if err == nil {
		err = os.MkdirAll(filepath.Dir(*output), 0755)
	}
	if err == nil {
		err = os.WriteFile(*output, append(data, '\n'), 0644)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
