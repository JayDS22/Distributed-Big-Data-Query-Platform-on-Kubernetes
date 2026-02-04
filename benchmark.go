package cmd

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"time"

	"github.com/spf13/cobra"
)

var (
	benchEngine  string
	benchQuery   string
	scale        string
	iterations   int
	outputFormat string
)

var BenchmarkCmd = &cobra.Command{
	Use:   "benchmark",
	Short: "Run benchmark queries",
	Long:  "Execute TPC-H or TPC-DS queries against Trino/Spark and capture performance metrics",
	RunE: func(cmd *cobra.Command, args []string) error {
		if benchEngine == "" {
			return fmt.Errorf("--engine flag is required (trino or spark)")
		}

		log.Printf("Running benchmark: %s on %s\n", benchQuery, benchEngine)
		log.Printf("Scale: %s, Iterations: %d, Format: %s\n", scale, iterations, outputFormat)

		// Call Python benchmarking suite
		return runPythonBenchmark(cmd.Context(), benchEngine, benchQuery, scale, iterations, outputFormat)
	},
}

func runPythonBenchmark(ctx context.Context, engine, query, scale string, iterations int, format string) error {
	pythonScript := "python/benchmark_runner.py"

	args := []string{
		pythonScript,
		"--engine", engine,
		"--query", query,
		"--scale", scale,
		"--iterations", fmt.Sprintf("%d", iterations),
		"--output-format", format,
	}

	cmd := exec.CommandContext(ctx, "python3", args...)

	log.Printf("Executing: python3 %v\n", args)

	// Run command and capture output
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("benchmark failed: %w\nOutput: %s", err, string(output))
	}

	fmt.Println(string(output))

	log.Printf("✓ Benchmark completed successfully\n")
	log.Printf("Results saved in %s format\n", format)

	return nil
}

func init() {
	BenchmarkCmd.Flags().StringVar(&benchEngine, "engine", "", "Query engine (trino or spark)")
	BenchmarkCmd.Flags().StringVar(&benchQuery, "query", "tpch-q1", "Query to run (e.g., tpch-q1, tpch-q22, tpcds-q1)")
	BenchmarkCmd.Flags().StringVar(&scale, "scale", "1", "TPC scale factor (1, 10, 100, etc.)")
	BenchmarkCmd.Flags().IntVar(&iterations, "iterations", 3, "Number of iterations")
	BenchmarkCmd.Flags().StringVar(&outputFormat, "output", "json", "Output format (json or csv)")

	BenchmarkCmd.MarkFlagRequired("engine")
}
