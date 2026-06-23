package main

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/amiv1/lsq/system"
	"github.com/spf13/cobra"
)

var todayCmd = &cobra.Command{
	Use:          "today [text...]",
	Short:        "Open today's journal, or append text to it",
	Aliases:      []string{"t"},
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runJournal("", args)
	},
}

var yesterdayCmd = &cobra.Command{
	Use:          "yesterday [text...]",
	Short:        "Open yesterday's journal, or append text to it",
	Aliases:      []string{"y"},
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		date := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
		return runJournal(date, args)
	},
}

var agoCmd = &cobra.Command{
	Use:          "ago <n> [text...]",
	Short:        "Open journal from N days ago, or append text to it",
	Aliases:      []string{"a"},
	Args:         cobra.MinimumNArgs(1),
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		n, err := strconv.Atoi(args[0])
		if err != nil || n < 0 {
			return fmt.Errorf("expected a non-negative integer for <n>, got %q", args[0])
		}
		date := time.Now().AddDate(0, 0, -n).Format("2006-01-02")
		return runJournal(date, args[1:])
	},
}

func init() {
	rootCmd.Flags().IntVarP(&indentFlag, "indent", "i", 0, "Logseq nesting level for appended text (1=child, 2=grandchild, etc.)")
	for _, cmd := range []*cobra.Command{todayCmd, yesterdayCmd, agoCmd} {
		cmd.Flags().IntVarP(&indentFlag, "indent", "i", 0, "Logseq nesting level for appended text (1=child, 2=grandchild, etc.)")
	}
}

// runJournal handles open/append logic shared by today, yesterday, ago, and root.
// specDate is "" for today. args is the text to append; empty = open in editor.
func runJournal(specDate string, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	if err := checkDirExists(cfg.DirPath); err != nil {
		return err
	}
	journalPath, err := system.GetJournal(cfg, cfg.JournalsDir, specDate)
	if err != nil {
		return fmt.Errorf("error resolving journal path: %w", err)
	}
	if len(args) == 0 {
		if stdinPiped() {
			content, readErr := io.ReadAll(os.Stdin)
			if readErr != nil {
				return fmt.Errorf("error reading STDIN: %w", readErr)
			}
			return system.AppendToFile(journalPath, string(content), indentTabs())
		}
		system.LoadEditor(editorFlag, journalPath)
		return nil
	}
	return system.AppendToFile(journalPath, strings.Join(args, " "), indentTabs())
}

// checkDirExists returns an error if the Logseq directory does not exist.
func checkDirExists(dirPath string) error {
	if _, err := os.Stat(dirPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("could not find Logseq files at %q.\nMake sure the path is correct and the directories exist", dirPath)
		}
		return fmt.Errorf("error loading the main directory: %w", err)
	}
	return nil
}

// stdinPiped returns true when STDIN is connected to a pipe rather than a terminal.
func stdinPiped() bool {
	stat, err := os.Stdin.Stat()
	return err == nil && (stat.Mode()&os.ModeCharDevice) == 0
}

// indentTabs converts the 1-indexed nesting level in indentFlag to a tab count.
// Level 0 (default) means no indent; level 1 = 1 tab, level 2 = 2 tabs, etc.
func indentTabs() int {
	if indentFlag <= 0 {
		return 0
	}
	return indentFlag
}
