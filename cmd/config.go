package cmd

import (
	"fmt"

	"doc-to-text/pkg/config"

	"github.com/spf13/cobra"
)

// configCmd represents the config command
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage tool path configuration",
	Long: `Manage tool path configuration settings.

Configuration is stored in a JSON file in your user configuration directory (~/.doc-to-text/config.json).
You can list all tool paths, get specific values, or set new values.

Available commands:
  list  - List all configured tool paths
  get   - Get a specific tool path
  set   - Set a specific tool path

Examples:
  doc-to-text config list                              # List all tool paths
  doc-to-text config get surya_ocr_path               # Get Surya OCR path
  doc-to-text config set surya_ocr_path /usr/bin/surya_ocr  # Set Surya OCR path
  doc-to-text config set calibre_path /opt/calibre/ebook-convert  # Set Calibre path`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		switch args[0] {
		case "list":
			listConfig()
		case "get":
			if len(args) < 2 {
				fmt.Println("Error: 'get' command requires a key name")
				fmt.Println("Usage: doc-to-text config get <key>")
				return
			}
			getConfig(args[1])
		case "set":
			if len(args) < 3 {
				fmt.Println("Error: 'set' command requires a key and value")
				fmt.Println("Usage: doc-to-text config set <key> <value>")
				return
			}
			setConfig(args[1], args[2])
		default:
			fmt.Printf("Error: Unknown config command '%s'\n", args[0])
			fmt.Println("Available commands: list, get, set")
		}
	},
}

// listConfig lists all tool path configuration settings
func listConfig() {
	fmt.Println("🛠️  Tool Path Configuration")
	fmt.Println("===========================")

	// Load current config
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Printf("❌ Error loading configuration: %v\n", err)
		return
	}

	// Show config file location
	configPath, _ := config.GetConfigFilePath()
	fmt.Printf("📁 Config file: %s\n\n", configPath)

	// Display tool paths
	fmt.Println("🛠️  Tool Paths:")
	fmt.Printf("  %-22s = %s\n", "llm_caller_path", getDisplayValue(cfg.LLMCallerPath))
	fmt.Printf("  %-22s = %s\n", "surya_ocr_path", getDisplayValue(cfg.SuryaOCRPath))
	fmt.Printf("  %-22s = %s\n", "calibre_path", getDisplayValue(cfg.CalibrePath))
	fmt.Printf("  %-22s = %s\n", "pandoc_path", getDisplayValue(cfg.PandocPath))
	fmt.Printf("  %-22s = %s\n", "ghostscript_path", getDisplayValue(cfg.GhostscriptPath))

	fmt.Println("\n💡 Tip: Use 'doc-to-text config get <key>' to get specific values")
	fmt.Println("💡 Tip: Use 'doc-to-text config set <key> <value>' to change tool paths")
	fmt.Println("💡 Note: Other settings (OCR tool, concurrency, etc.) are runtime-only")
}

// getConfig gets a specific configuration value
func getConfig(key string) {
	value, err := config.GetConfigValue(key)
	if err != nil {
		fmt.Printf("❌ Error getting config value '%s': %v\n", key, err)
		return
	}

	fmt.Printf("📝 %s = %v\n", key, value)
}

// setConfig sets a specific configuration value
func setConfig(key, value string) {
	// Set the value (all config values are strings - tool paths)
	err := config.SetConfigValue(key, value)
	if err != nil {
		fmt.Printf("❌ Error setting config value '%s': %v\n", key, err)
		return
	}

	fmt.Printf("✅ Successfully set %s = %v\n", key, value)
	fmt.Printf("💡 Tip: Make sure the tool is installed and accessible at this path\n")
}

// getDisplayValue returns a display-friendly value for empty strings
func getDisplayValue(value string) string {
	if value == "" {
		return "(not set)"
	}
	return value
}

// configListCmd represents the 'config list' command
var configListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all tool path settings",
	Run: func(cmd *cobra.Command, args []string) {
		listConfig()
	},
}

// configGetCmd represents the 'config get' command
var configGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get a specific tool path value",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		getConfig(args[0])
	},
}

// configSetCmd represents the 'config set' command
var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a specific tool path value",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		setConfig(args[0], args[1])
	},
}

func init() {
	// Add config command to root
	rootCmd.AddCommand(configCmd)

	// Add subcommands to config
	configCmd.AddCommand(configListCmd)
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configSetCmd)
}
