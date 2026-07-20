package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/amiv1/lsq/config"
	"github.com/spf13/cobra"
)

const semVer = "2.4.0"

var (
	dirFlag    string
	editorFlag string
	indentFlag int
)

var rootCmd = &cobra.Command{
	Use:          "lsq",
	Short:        "The ultra-fast CLI companion for Logseq",
	Version:      semVer,
	SilenceUsage: true,
	Args:         cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runJournal("", args)
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&dirFlag, "dir", "", "Logseq directory (overrides config)")
	rootCmd.PersistentFlags().StringVar(&editorFlag, "editor", "", "Editor to use (default: $EDITOR, fallback: vim)")

	rootCmd.AddCommand(todayCmd, yesterdayCmd, agoCmd, pageCmd, getCmd, searchCmd, findCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// loadConfig loads lsq configuration and applies the --dir override if set.
func loadConfig() (*config.Config, error) {
	cfg, err := config.Load()
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("error loading configuration: %w", err)
	}
	if dirFlag != "" {
		expanded, expandErr := config.ExpandPath(dirFlag)
		if expandErr != nil {
			return nil, fmt.Errorf("error expanding directory path: %w", expandErr)
		}
		cfg.DirPath = expanded
		cfg.JournalsDir = filepath.Join(cfg.DirPath, "journals")
		cfg.PagesDir = filepath.Join(cfg.DirPath, "pages")
	}
	return cfg, nil
}

// resolveSearch parses a search query and returns a regex pattern.
// A value wrapped in leading and trailing slashes (e.g. /pattern/) is treated
// as a raw regex; anything else is matched literally.
func resolveSearch(query string) (prefix, regexPattern string) {
	if len(query) > 1 && strings.HasPrefix(query, "/") && strings.HasSuffix(query, "/") {
		return "", query[1 : len(query)-1]
	}
	return "", regexp.QuoteMeta(query)
}

// resolvePageName returns the filename in pagesDir matching name (case-insensitive,
// ignoring extension). Returns name unchanged if it already has an extension or no
// match is found.
func resolvePageName(pagesDir, name string) string {
	if filepath.Ext(name) != "" {
		return name
	}
	entries, err := os.ReadDir(pagesDir)
	if err != nil {
		return name
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			base := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
			if strings.EqualFold(base, name) {
				return entry.Name()
			}
		}
	}
	return name
}
