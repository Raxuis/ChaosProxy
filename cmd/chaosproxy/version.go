package main

import (
	"fmt"
	"io"
	"runtime/debug"
	"strings"
	"text/tabwriter"

	"github.com/Raxuis/chaosproxy/internal/profiles"
)

var version = "dev"

func versionString() string {
	if version != "dev" {
		return version
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return strings.TrimPrefix(info.Main.Version, "v")
	}
	return version
}

func printProfiles(w io.Writer) {
	table := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	for _, profile := range profiles.List() {
		fmt.Fprintf(table, "%s\t%s\n", profile.Name, profile.Description)
	}
	_ = table.Flush()
}
