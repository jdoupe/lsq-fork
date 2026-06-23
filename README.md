<!--<p align="center">
<img width="25%" src="https://lsq.sh/media/img/lsq_logo_cropped.png" alt="lsq logo">
</p>-->

# lsq

[![Go Report Card](https://goreportcard.com/badge/github.com/amiv1/lsq-fork)](https://goreportcard.com/report/github.com/amiv1/lsq-fork)
[![Release](https://img.shields.io/github/v/release/amiv1/lsq-fork)](https://github.com/amiv1/lsq-fork/releases)
[![License](https://img.shields.io/github/license/amiv1/lsq-fork)](https://github.com/amiv1/lsq-fork/blob/master/LICENSE)

The ultra-fast CLI companion for [Logseq](https://github.com/logseq/logseq) designed to speed up your note capture directly from the terminal!

> **Fork of [jrswab/lsq](https://github.com/jrswab/lsq)** — aimed at improving UX and adding more features.

## Why lsq?
- ⚡️ Lightning-fast journal additions without leaving your terminal
- ⌨️ Optimized for both quick captures and extended writing sessions
- 🎯 Native support for Logseq's file naming and formatting conventions
- 🔄 Seamless integration with your existing Logseq workflow
- 💻 Built by Logseq users, for Logseq users

## Features
- Just run `lsq` to open today's journal, or `lsq your note` to append instantly
- Browse your entire journal history with `lsq g j` — rendered Markdown, coloured `[[links]]` and `#tags`
- Open any page or journal entry in your `$EDITOR`
- Search across all notes by text or regex
- Works with both Markdown and Org Mode files
- Simple config file — set your Logseq directory once and forget it

## Installation

If you have the [original lsq](https://github.com/jrswab/lsq) installed with `go install`, remove it first:

```shell
rm $(which lsq)
```

### macOS — Homebrew

```shell
brew install amiv1/lsq/lsq
```

### Debian / Ubuntu

Add a repository:
```shell
curl -fsSL https://apt.fury.io/amiv1/gpg.key | sudo gpg --dearmor -o /etc/apt/trusted.gpg.d/fury-amiv1.gpg
echo "deb [signed-by=/etc/apt/trusted.gpg.d/fury-amiv1.gpg] https://apt.fury.io/amiv1/ * *" | sudo tee /etc/apt/sources.list.d/lsq-fork.list
```

Install lsq:
```shell
sudo apt update && sudo apt install lsq
```

### Fedora / RHEL / CentOS

Add a repository:
```shell
echo "[fury-amiv1]
name=lsq (Gemfury)
baseurl=https://yum.fury.io/amiv1/
enabled=1
gpgcheck=0" | sudo tee /etc/yum.repos.d/fury-amiv1.repo
```

Install lsq:
```shell
sudo dnf install lsq
```

### Build and install from source

Requires Go:

```bash
git clone https://github.com/amiv1/lsq-fork.git
cd lsq-fork
go install .
```

Make sure you have the location of the Go binaries in your `$PATH`. Run go env and find the variable called GOPATH.
Then copy that location to your shell's `$PATH` if it's not already there.

## Usage

### Commands

Each command has a short alias. All commands accept `--dir` and `--editor` as global flags.

#### Journal commands
Open the journal in your editor, or append text if provided.

`lsq [text...]` writes to today's journal by default — no subcommand needed. The explicit `lsq today` / `lsq t` form is also available.

| Command | Alias | Description |
|---------|-------|-------------|
| `lsq [text...]` | — | Today's journal (default) |
| `lsq today [text...]` | `lsq t` | Today's journal |
| `lsq yesterday [text...]` | `lsq y` | Yesterday's journal |
| `lsq ago <n> [text...]` | `lsq a <n>` | Journal from N days ago |

When piped (`echo "text" | lsq`), STDIN is appended automatically.  
Flag: `-i`/`--indent <n>` — indentation level for appended text.  
Tip: in shells, `#` starts a comment — use `@tag` in CLI input and `lsq` will write it as `#tag` (e.g. `lsq Met with Bob @work`).

#### `lsq page <name> [text...]` / `lsq p`
Open a specific page in your editor, or append text to it. File extension is auto-detected.

Flag: `-i`/`--indent <n>`

#### `lsq get` / `lsq g`
Print to STDOUT. Uses subcommands for date/target selection (default: today).

| Subcommand | Alias | Example |
|------------|-------|---------|
| *(default)* | — | `lsq g` |
| `get yesterday` | `g y` | `lsq g y` |
| `get ago <n>` | `g a <n>` | `lsq g a 3` |
| `get date <yyyy-MM-dd>` | `g d <date>` | `lsq g d 2024-01-15` |
| `get page <name>` | `g p <name>` | `lsq g p notes` |
| `get journal` | `g j` | `lsq g j` |

#### `lsq get journal` / `lsq g j`
Browse all journal entries in a pager (newest first) with full Markdown rendering, coloured `[[links]]` and `#tags`.

Flag: `--raw` — print raw Markdown without formatting or colours.

The pager defaults to `less -R`. Override with `LSQ_PAGER`:
```bash
LSQ_PAGER="bat" lsq g j
```

#### `lsq search <query>` / `lsq s`
Search file **contents** across all journals and pages. Plain text matches literally; wrap in `/…/` for regex.

Flag: `-o`/`--open` — open the first matching file in editor.

#### `lsq find <prefix>` / `lsq f`
Search page **filenames** by prefix.

Flag: `-o`/`--open` — open the first matching page in editor.

#### Global flags
Available on all commands:
- `--dir <path>` — Logseq directory (overrides config). Supports `~` and environment variables.
- `--editor <cmd>` — Editor to use (default: `$EDITOR`, fallback: vim).

#### Environment variables
- `LSQ_PAGER` — Pager used by `lsq g j` (default: `less -R`). Example: `LSQ_PAGER=bat`.

### Configuration File
This file must be stored in your config directory as `lsq/config.edn`.
On Unix systems, it returns `$XDG_CONFIG_HOME` if non-empty, else `$HOME/.config` will be used.
On macOS, it first looks in `$HOME/Library/Application Support/lsq/config.edn`. If not found, it falls back to `$HOME/.config/lsq/config.edn`.
On Windows, it returns `%AppData%`.
On Plan 9, it returns `$home/lib`.

#### Configuration Behavior
The configuration file will override any lsq defaults which are defined. If a CLI flag is provided, the flag value will override the config file value.

#### Configuration File Example:
```EDN
{
  ;; Either "Markdown" or "Org".
  :file/type "Markdown"
  ;; This will be used for journal file names
  ;; Using the format below and the file type above will produce 2025.01.01.md
  :file/format "yyyy_MM_dd"
  ;; The directory which holds all your notes
  ;; Supports ~ and environment variables (e.g., ~/Logseq or $HOME/Logseq)
  :directory "~/Logseq"
}
```
**Note:** The configured directory must contain both a `journals` and `pages` subdirectory for lsq to function properly. These are automatically created when using Logseq, but will need to be manually created if setting lsq to use a new directory or without Logseq.

### Usage Examples:

Opens today's journal in `$EDITOR`.
```shell
lsq
```

Appends `Discussed Q2 planning @work` as a bullet point to today's journal.
```shell
lsq Discussed Q2 planning @work
```

Opens today's journal in `$EDITOR` (explicit form).
```shell
lsq t
```

Appends to today's journal (explicit form).
```shell
lsq t Read article on [[Clojure]] macros @learning
```

Appends to the journal from 2 days ago.
```shell
lsq a 2 Grocery run @personal
```

Appends to yesterday's journal.
```shell
lsq y Morning standup notes @work
```

Appends to the page named `my-page` (extension auto-detected). Without text, opens the page in editor.
```shell
lsq p my-page Notes for [[Project X]] @work
```

Appends text as an indented bullet (one tab level deep). Use `--indent 2` for two levels, and so on.
```shell
lsq --indent 1 TODO Follow up with Alice @work
```

Searches all journals and pages for lines containing `TODO`.
```shell
lsq search TODO
```

Searches using the regex `TODO|FIXME`.
```shell
lsq search '/TODO|FIXME/'
```

Searches for `TODO` and opens the first matching file in editor.
```shell
lsq search TODO --open
```

Lists pages whose filename starts with `go`.
```shell
lsq find go
```

Prints today's journal to STDOUT. Useful for shell integration, piping, or display widgets.
```shell
lsq g
```

Prints the journal from 3 days ago to STDOUT.
```shell
lsq g a 3
```

Prints the `notes` page to STDOUT.
```shell
lsq g p notes
```

Opens an interactive pager showing all journal entries newest first, with Markdown rendering and coloured `[[links]]` and `#tags`.
```shell
lsq g j
```

Same, but prints raw Markdown without any formatting.
```shell
lsq g j --raw
```

Appends the contents of `~/.zshrc` to today's journal via STDIN.
```shell
cat ~/.zshrc | lsq
```

Appends STDOUT and STDERR of a long-running job to a new page.
```shell
run_long_batch_job |& lsq p "long-job.$(date +%s).log"
```

## Contributing
For information on contributing to lsq check out [CONTRIBUTING.md](https://github.com/amiv1/lsq-fork/blob/master/CONTRIBUTING.md).
