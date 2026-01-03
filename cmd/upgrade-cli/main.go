package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/spf13/cobra"
)

var (
	apiEndpoint  string
	outputFormat string
)

type UpgradeStatus struct {
	CurrentPhase int                    `json:"currentPhase"`
	Status       string                 `json:"status"`
	Metadata     map[string]interface{} `json:"metadata"`
}

type HealthResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

func main() {
	rootCmd := &cobra.Command{
		Use:   "upgrade-cli",
		Short: "CLI tool for managing consensus upgrades",
	}

	rootCmd.PersistentFlags().StringVarP(&apiEndpoint, "endpoint", "e", "http://localhost:8080", "API endpoint")
	rootCmd.PersistentFlags().StringVarP(&outputFormat, "output", "o", "json", "Output format (json)")

	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "Get upgrade status",
		RunE:  runStatus,
	}

	healthCmd := &cobra.Command{
		Use:   "health",
		Short: "Check API health",
		RunE:  runHealth,
	}

	rootCmd.AddCommand(statusCmd, healthCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runStatus(cmd *cobra.Command, args []string) error {
	resp, err := http.Get(apiEndpoint + "/api/upgrade/status")
	if err != nil {
		return fmt.Errorf("failed to get status: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var status UpgradeStatus
	if err := json.Unmarshal(body, &status); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	formatted, _ := json.MarshalIndent(status, "", "  ")
	fmt.Println(string(formatted))
	return nil
}

func runHealth(cmd *cobra.Command, args []string) error {
	resp, err := http.Get(apiEndpoint + "/api/upgrade/health")
	if err != nil {
		return fmt.Errorf("failed to check health: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var health HealthResponse
	if err := json.Unmarshal(body, &health); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	formatted, _ := json.MarshalIndent(health, "", "  ")
	fmt.Println(string(formatted))
	return nil
}
