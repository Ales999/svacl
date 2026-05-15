package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"

	"github.com/alecthomas/kong"
)

// Flags for app:
var cli struct {
	Config     string `arg:"" required:"" name:"CONFIG" help:"Cisco config file name or path"`
	Debug      bool   `help:"Enable debug output" short:"d"`
	Quiet      bool   `help:"Lite mode — one ACL name per line (active SVI only)" short:"q"`
	UniqueAcls bool   `help:"Remove duplicate ACL names (only with -q)"`
	CfgDir     string `required:"" help:"Path to backup cisco files" env:"CISCONFS" type:"existingdir"`
}

func main() {
	ctx := kong.Parse(&cli,
		kong.Name("svlacl"),
		kong.Description("List ACLs applied to SVI interfaces from Cisco config files"),
		kong.UsageOnError(),
	)

	configPath := cli.Config
	if !filepath.IsAbs(configPath) && cli.CfgDir != "" {
		configPath = filepath.Join(cli.CfgDir, configPath)
	}

	var err error
	configPath, err = filepath.Abs(configPath)
	if err != nil {
		ctx.FatalIfErrorf(err)
	}

	if !checkTextFile(configPath) {
		fmt.Fprintf(os.Stderr, "Error: %s is not a valid text file.\n", configPath)
		os.Exit(1)
	}

	results, err := ParseSVIAclFile(configPath)
	ctx.FatalIfErrorf(err)

	if cli.Debug {
		log.Printf("Config: %s (%d SVI found)\n", configPath, len(results))
	}

	if cli.Quiet {
		printLite(results, cli.UniqueAcls)
	} else {
		printTable(results)
	}
	os.Exit(0)
}

func printTable(results []SVIAclInfo) {
	if len(results) == 0 {
		fmt.Println("No SVI interfaces found.")
		return
	}

	hostname := ""
	for _, r := range results {
		if r.Hostname != "" {
			hostname = r.Hostname
			break
		}
	}
	if hostname != "" {
		fmt.Printf("Hostname: %s\n", hostname)
	}

	for _, r := range results {
		status := "up"
		if r.Shutdown {
			status = "DOWN"
		}

		fmt.Printf("%-12s | IP: %-21s | VRF: %-8s | Status: %-4s ACL In: %-10s | ACL Out: %s\n",
			r.VlanName,
			r.IPAddr,
			r.VRF,
			status,
			r.ACLIn,
			r.ACLOut,
		)
	}
}

func printLite(results []SVIAclInfo, unique bool) {
	var acls []string
	for _, r := range results {
		if r.Shutdown {
			continue
		}
		if r.ACLIn != "" {
			acls = append(acls, r.ACLIn)
		}
		if r.ACLOut != "" {
			acls = append(acls, r.ACLOut)
		}
	}
	if unique {
		seen := make(map[string]bool)
		var deduped []string
		for _, a := range acls {
			if !seen[a] {
				seen[a] = true
				deduped = append(deduped, a)
			}
		}
		acls = deduped
	}
	sort.Strings(acls)
	for _, a := range acls {
		fmt.Println(a)
	}
}

func checkTextFile(filePath string) bool {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return false
	}
	for _, b := range data {
		if b == 0 {
			return false
		}
	}
	return true
}
