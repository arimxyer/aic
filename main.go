package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var version = "dev"

var rootCmd = &cobra.Command{
	Use:     "aic",
	Short:   "AI Coding Agent Changelog Viewer",
	Long:    "Fetch the latest changelogs for popular AI coding assistants.",
	Version: version,
}

var latestCmd = &cobra.Command{
	Use:     "latest",
	Short:   "Show releases from all sources in last 24h",
	GroupID: "commands",
	RunE: func(cmd *cobra.Command, args []string) error {
		webOpen, _ := cmd.Flags().GetBool("web")
		if webOpen {
			for _, src := range enabledSources() {
				openBrowser(src.URL())
			}
			return nil
		}
		jsonOutput, _ := cmd.Flags().GetBool("json")
		runLatestCommand(jsonOutput)
		return nil
	},
}

var statusCmd = &cobra.Command{
	Use:     "status",
	Short:   "Show status table of all sources",
	GroupID: "commands",
	RunE: func(cmd *cobra.Command, args []string) error {
		webOpen, _ := cmd.Flags().GetBool("web")
		if webOpen {
			for _, src := range enabledSources() {
				openBrowser(src.URL())
			}
			return nil
		}
		jsonOutput, _ := cmd.Flags().GetBool("json")
		runStatusCommand(jsonOutput)
		return nil
	},
}

var configCmd = &cobra.Command{
	Use:     "config",
	Short:   "Configure which sources to show",
	GroupID: "commands",
	Run: func(cmd *cobra.Command, args []string) {
		runConfigCommand()
	},
}

var listSourcesCmd = &cobra.Command{
	Use:     "list-sources",
	Short:   "List all available sources",
	GroupID: "commands",
	Run: func(cmd *cobra.Command, args []string) {
		for name, src := range sources {
			fmt.Printf("  %s\t%s\n", name, src.DisplayName)
		}
	},
}

func createSourceCommand(name string, src Source) *cobra.Command {
	cmd := &cobra.Command{
		Use:     name,
		Short:   fmt.Sprintf("View %s changelog", src.DisplayName),
		GroupID: "sources",
		RunE: func(cmd *cobra.Command, args []string) error {
			webOpen, _ := cmd.Flags().GetBool("web")
			if webOpen {
				openBrowser(src.URL())
				return nil
			}

			entries, err := src.Fetch()
			if err != nil {
				return fmt.Errorf("fetching changelog: %w", err)
			}

			if len(entries) == 0 {
				return fmt.Errorf("no changelog entries found")
			}

			pickVersion, _ := cmd.Flags().GetBool("pick")
			if pickVersion {
				runPickCommand(src.DisplayName, entries)
				return nil
			}

			listVersions, _ := cmd.Flags().GetBool("list")
			if listVersions {
				outputVersionList(src.DisplayName, entries)
				return nil
			}

			targetVersion, _ := cmd.Flags().GetString("version")
			var entry *ChangelogEntry
			if targetVersion != "" {
				for i := range entries {
					if entries[i].Version == targetVersion {
						entry = &entries[i]
						break
					}
				}
				if entry == nil {
					return fmt.Errorf("version %s not found", targetVersion)
				}
			} else {
				entry = &entries[0]
			}

			jsonOutput, _ := cmd.Flags().GetBool("json")
			mdOutput, _ := cmd.Flags().GetBool("md")

			if jsonOutput {
				outputJSON(entry)
			} else if mdOutput {
				outputMarkdown(entry)
			} else {
				outputRendered(src.DisplayName, entry)
			}

			return nil
		},
	}

	cmd.Flags().BoolP("json", "j", false, "Output as JSON")
	cmd.Flags().BoolP("md", "m", false, "Output as markdown")
	cmd.Flags().BoolP("list", "l", false, "List all versions")
	cmd.Flags().BoolP("pick", "p", false, "Interactive version picker")
	cmd.Flags().String("version", "", "Get specific version")
	cmd.Flags().BoolP("web", "w", false, "Open in browser")

	cmd.MarkFlagsMutuallyExclusive("json", "md")
	cmd.MarkFlagsMutuallyExclusive("list", "pick")
	cmd.MarkFlagsMutuallyExclusive("list", "version")
	cmd.MarkFlagsMutuallyExclusive("pick", "version")

	return cmd
}

func init() {
	// Register command groups
	rootCmd.AddGroup(
		&cobra.Group{ID: "sources", Title: "Sources:"},
		&cobra.Group{ID: "commands", Title: "Commands:"},
	)

	// Add flags to latest and status commands
	latestCmd.Flags().BoolP("json", "j", false, "Output as JSON")
	latestCmd.Flags().BoolP("web", "w", false, "Open in browser")

	statusCmd.Flags().BoolP("json", "j", false, "Output as JSON")
	statusCmd.Flags().BoolP("web", "w", false, "Open in browser")

	// Register fixed commands
	rootCmd.AddCommand(latestCmd, statusCmd, configCmd, listSourcesCmd)

	// Dynamically register source commands
	for name, src := range sources {
		rootCmd.AddCommand(createSourceCommand(name, src))
	}
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
