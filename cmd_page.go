package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/amiv1/lsq/config"
	"github.com/amiv1/lsq/system"
	"github.com/spf13/cobra"
)

var pageCmd = &cobra.Command{
	Use:          "page <name> [text...]",
	Short:        "Open a page, or append text to it",
	Aliases:      []string{"p"},
	Args:         cobra.MinimumNArgs(1),
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPage(args[0], args[1:])
	},
}

func init() {
	pageCmd.Flags().IntVarP(&indentFlag, "indent", "i", 0, "Logseq nesting level for appended text (1=child, 2=grandchild, etc.)")
}

func runPage(name string, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	pagePath := resolvePagePath(cfg, name)
	if len(args) == 0 {
		if stdinPiped() {
			content, readErr := io.ReadAll(os.Stdin)
			if readErr != nil {
				return fmt.Errorf("error reading STDIN: %w", readErr)
			}
			return system.AppendToFile(pagePath, string(content), indentTabs())
		}
		system.LoadEditor(editorFlag, pagePath)
		return nil
	}
	return system.AppendToFile(pagePath, strings.Join(args, " "), indentTabs())
}

// resolvePagePath returns the full path of a page, auto-detecting the file extension.
func resolvePagePath(cfg *config.Config, name string) string {
	return filepath.Join(cfg.PagesDir, resolvePageName(cfg.PagesDir, name))
}
