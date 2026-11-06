package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/OdairPianta/unisinos_usp_api/tests/benchmark"
)

func split(value string) []string {
	if value == "" {
		return nil
	}
	return strings.Split(value, ",")
}
func main() {
	mode := flag.String("mode", "deterministic", "deterministic or gemini (paid, requires explicit environment opt-in)")
	runs := flag.Int("runs", 3, "repetitions for every selected case")
	ids := flag.String("case", "", "comma-separated exact case IDs")
	tags := flag.String("tag", "", "comma-separated tags (any matching tag)")
	output := flag.String("output", filepath.Join(os.TempDir(), "onlife-benchmark-"+time.Now().UTC().Format("20060102T150405")+".json"), "protected JSON report path")
	list := flag.Bool("list", false, "list scenarios without database or provider calls")
	flag.Parse()
	cases, err := benchmark.LoadCases()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	cases, err = benchmark.SelectCases(cases, split(*ids), split(*tags))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	if *list {
		for _, c := range cases {
			fmt.Printf("%s [%s]\n", c.ID, strings.Join(c.Tags, ","))
		}
		return
	}
	if *mode != "deterministic" && *mode != "gemini" || *runs < 1 || *runs > 20 {
		fmt.Fprintln(os.Stderr, "Invalid mode/run count")
		os.Exit(2)
	}
	if *mode == "gemini" && os.Getenv("ONLIFE_BENCHMARK_ALLOW_PROVIDER") != "1" {
		fmt.Fprintln(os.Stderr, "Explicit provider opt-in required; see README")
		os.Exit(2)
	}
	if err := benchmark.SeedFixtures(); err != nil {
		fmt.Fprintln(os.Stderr, "Cannot prepare isolated fixtures:", err)
		os.Exit(2)
	}
	report, err := benchmark.Run(context.Background(), cases, benchmark.Options{Mode: *mode, Runs: *runs, Progress: func(result benchmark.CaseResult) {
		status := "PASS"
		if !result.Passed {
			status = "FAIL"
		}
		fmt.Printf("%s run=%d %s\n", result.ID, result.Run, status)
	}})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	payload, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, "Cannot encode report")
		os.Exit(2)
	}
	destination := filepath.Clean(*output)
	if err = os.MkdirAll(filepath.Dir(destination), 0700); err != nil {
		fmt.Fprintln(os.Stderr, "Cannot prepare report directory")
		os.Exit(2)
	}
	file, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Cannot create report")
		os.Exit(2)
	}
	file.Chmod(0600)
	_, err = file.Write(append(payload, '\n'))
	closeErr := file.Close()
	if err != nil || closeErr != nil {
		fmt.Fprintln(os.Stderr, "Cannot save complete report")
		os.Exit(2)
	}
	fmt.Printf("mode=%s cases=%d runs=%d passed=%d failed=%d report=%s\n", report.Mode, len(cases), report.Runs, report.Passed, report.Failed, destination)
	if report.Failed > 0 {
		os.Exit(1)
	}
}
