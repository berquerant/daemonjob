// Copyright 2026 berquerant.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/berquerant/daemonjob/internal/daemonjobctl"
	"sigs.k8s.io/yaml"
)

const (
	defaultBroadcastImage = "ghcr.io/berquerant/daemonjob-broadcast:latest"
	defaultClusterRole    = "daemonjob-broadcast-role"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "skeleton":
		runSkeleton(os.Args[2:])
	case "generate":
		runGenerate(os.Args[2:])
	case "-h", "--help", "help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "daemonjobctl: unknown command %q\n\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `daemonjobctl - CLI helper for daemonjob custom resources

Usage:
  daemonjobctl <command> [flags]

Commands:
  skeleton <kind>   Output a skeleton YAML for the given custom resource kind.
                    Supported kinds: daemonjob | daemoncronjob | daemoncronjobset

  generate          Generate the Kubernetes manifests that daemonjob would create
                    for a given custom resource and a list of node names.
                    For DaemonJob and DaemonCronJob, broadcast resources are output
                    directly, and worker Jobs are output as commented-out YAML.
                    For DaemonCronJobSet, all per-node CronJobs are output directly.

Flags for 'generate':
  -f string
        Input manifest YAML file (reads from stdin if omitted)
  -nodes string
        Comma-separated list of node names (required)
  -broadcast-image string
        Broadcast container image used for DaemonJob/DaemonCronJob (default: %s)
  -broadcast-role string
        Broadcast ClusterRole name used for DaemonJob/DaemonCronJob (default: %s)

Examples:
  daemonjobctl skeleton daemonjob > my-daemonjob.yaml
  daemonjobctl generate -f my-daemonjob.yaml -nodes node1,node2,node3
  cat my-daemonjob.yaml | daemonjobctl generate -nodes node1,node2
`, defaultBroadcastImage, defaultClusterRole)
}

func runSkeleton(args []string) {
	fs := flag.NewFlagSet("skeleton", flag.ExitOnError)
	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "skeleton: %v\n", err)
		os.Exit(1)
	}

	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "skeleton: missing required argument: kind")
		fmt.Fprintln(os.Stderr, "  usage: daemonjobctl skeleton <daemonjob|daemoncronjob|daemoncronjobset>")
		os.Exit(1)
	}

	kind := strings.ToLower(fs.Arg(0))

	var obj any
	switch kind {
	case "daemonjob":
		obj = daemonjobctl.DaemonJobSkeleton()
	case "daemoncronjob":
		obj = daemonjobctl.DaemonCronJobSkeleton()
	case "daemoncronjobset":
		obj = daemonjobctl.DaemonCronJobSetSkeleton()
	default:
		fmt.Fprintf(os.Stderr, "skeleton: unknown kind %q\n", kind)
		fmt.Fprintln(os.Stderr, "  supported: daemonjob, daemoncronjob, daemoncronjobset")
		os.Exit(1)
	}

	b, err := yaml.Marshal(obj)
	if err != nil {
		fmt.Fprintf(os.Stderr, "skeleton: marshal error: %v\n", err)
		os.Exit(1)
	}
	if _, err := os.Stdout.Write(b); err != nil {
		fmt.Fprintf(os.Stderr, "skeleton: write error: %v\n", err)
		os.Exit(1)
	}
}

func runGenerate(args []string) {
	fs := flag.NewFlagSet("generate", flag.ExitOnError)
	file := fs.String("f", "", "Input manifest YAML file (reads from stdin if omitted)")
	nodes := fs.String("nodes", "", "Comma-separated list of node names (required)")
	image := fs.String("broadcast-image", defaultBroadcastImage, "Broadcast container image")
	role := fs.String("broadcast-role", defaultClusterRole, "Broadcast ClusterRole name")

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "generate: %v\n", err)
		os.Exit(1)
	}

	if *nodes == "" {
		fmt.Fprintln(os.Stderr, "generate: -nodes is required")
		os.Exit(1)
	}

	// Split and trim node names.
	parts := strings.Split(*nodes, ",")
	nodeList := make([]string, 0, len(parts))
	for _, p := range parts {
		if n := strings.TrimSpace(p); n != "" {
			nodeList = append(nodeList, n)
		}
	}
	if len(nodeList) == 0 {
		fmt.Fprintln(os.Stderr, "generate: -nodes must contain at least one node name")
		os.Exit(1)
	}

	// Read manifest.
	var raw []byte
	var err error
	if *file != "" {
		raw, err = os.ReadFile(*file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "generate: read file %q: %v\n", *file, err)
			os.Exit(1)
		}
	} else {
		raw, err = io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintf(os.Stderr, "generate: read stdin: %v\n", err)
			os.Exit(1)
		}
	}

	result, err := daemonjobctl.Generate(raw, nodeList, *image, *role)
	if err != nil {
		fmt.Fprintf(os.Stderr, "generate: %v\n", err)
		os.Exit(1)
	}

	if err := daemonjobctl.WriteYAML(os.Stdout, result); err != nil {
		fmt.Fprintf(os.Stderr, "generate: write: %v\n", err)
		os.Exit(1)
	}
}
