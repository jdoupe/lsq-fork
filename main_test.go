package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// binaryPath holds the path to the compiled test binary.
var binaryPath string

// TestMain builds the lsq binary once, runs all tests, then cleans up.
func TestMain(m *testing.M) {
	tmp, err := os.MkdirTemp("", "lsq-test-bin-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create temp dir for binary: %v\n", err)
		os.Exit(1)
	}

	binaryPath = filepath.Join(tmp, "lsq")
	if out, buildErr := exec.Command("go", "build", "-o", binaryPath, ".").CombinedOutput(); buildErr != nil {
		fmt.Fprintf(os.Stderr, "failed to build lsq: %v\n%s\n", buildErr, out)
		os.RemoveAll(tmp)
		os.Exit(1)
	}

	code := m.Run()
	os.RemoveAll(tmp)
	os.Exit(code)
}

// --- Unit tests for helper functions ---

func TestResolveSearch(t *testing.T) {
	tests := map[string]struct {
		query      string
		wantPrefix string
		wantRegex  string
	}{
		"plain text becomes literal regex": {query: "keyword", wantPrefix: "", wantRegex: "keyword"},
		"special chars are escaped":        {query: "a.b", wantPrefix: "", wantRegex: `a\.b`},
		"regex with slashes":               {query: "/pattern/", wantPrefix: "", wantRegex: "pattern"},
		"regex complex pattern":            {query: "/TODO|FIXME/", wantPrefix: "", wantRegex: "TODO|FIXME"},
		"regex with special chars":         {query: "/^- \\w+/", wantPrefix: "", wantRegex: "^- \\w+"},
		"single slash not regex":           {query: "/", wantPrefix: "", wantRegex: "/"},
		"double slash is empty regex":      {query: "//", wantPrefix: "", wantRegex: ""},
		"starts with slash no end slash":   {query: "/noend", wantPrefix: "", wantRegex: "/noend"},
		"ends with slash no start slash":   {query: "nostart/", wantPrefix: "", wantRegex: "nostart/"},
		"empty query":                      {query: "", wantPrefix: "", wantRegex: ""},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			gotPrefix, gotRegex := resolveSearch(tc.query)
			if gotPrefix != tc.wantPrefix {
				t.Errorf("prefix: got %q, want %q", gotPrefix, tc.wantPrefix)
			}
			if gotRegex != tc.wantRegex {
				t.Errorf("regex: got %q, want %q", gotRegex, tc.wantRegex)
			}
		})
	}
}

func TestResolvePageName(t *testing.T) {
	pagesDir := t.TempDir()

	pageFiles := []string{"my-page.md", "Another Page.md", "notes.org", "UPPER.md"}
	for _, f := range pageFiles {
		if err := os.WriteFile(filepath.Join(pagesDir, f), []byte("content"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	tests := map[string]struct {
		input string
		want  string
	}{
		"exact name without extension":     {input: "my-page", want: "my-page.md"},
		"case-insensitive lowercase input": {input: "MY-PAGE", want: "my-page.md"},
		"case-insensitive uppercase file":  {input: "upper", want: "UPPER.md"},
		"name with spaces":                 {input: "Another Page", want: "Another Page.md"},
		"org file":                         {input: "notes", want: "notes.org"},
		"name already has extension":       {input: "my-page.md", want: "my-page.md"},
		"unknown extension kept as-is":     {input: "my-page.txt", want: "my-page.txt"},
		"no match falls back to input":     {input: "nonexistent", want: "nonexistent"},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := resolvePageName(pagesDir, tc.input)
			if got != tc.want {
				t.Errorf("resolvePageName(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestResolvePageNameMissingDir(t *testing.T) {
	got := resolvePageName("/no/such/dir", "page")
	if got != "page" {
		t.Errorf("expected fallback to input, got %q", got)
	}
}

// --- CLI integration tests ---

type cliEnv struct {
	logseqDir   string
	journalsDir string
	pagesDir    string
	env         []string
}

func newCLIEnv(t *testing.T) *cliEnv {
	t.Helper()
	tmp := t.TempDir()

	journalsDir := filepath.Join(tmp, "Logseq", "journals")
	pagesDir := filepath.Join(tmp, "Logseq", "pages")
	configDir := filepath.Join(tmp, ".config", "lsq")

	for _, d := range []string{journalsDir, pagesDir, configDir} {
		if err := os.MkdirAll(d, 0755); err != nil {
			t.Fatalf("mkdir %s: %v", d, err)
		}
	}

	env := append(os.Environ(),
		"HOME="+tmp,
		"XDG_CONFIG_HOME="+filepath.Join(tmp, ".config"),
		"EDITOR=true",
	)

	return &cliEnv{
		logseqDir:   filepath.Join(tmp, "Logseq"),
		journalsDir: journalsDir,
		pagesDir:    pagesDir,
		env:         env,
	}
}

func (e *cliEnv) run(args ...string) (stdout, stderr string, err error) {
	cmd := exec.Command(binaryPath, args...)
	cmd.Env = e.env

	var outBuf, errBuf strings.Builder
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	err = cmd.Run()
	return outBuf.String(), errBuf.String(), err
}

func (e *cliEnv) todayJournalPath() string {
	return filepath.Join(e.journalsDir, time.Now().Format("2006_01_02")+".md")
}

func (e *cliEnv) nDaysAgoJournalPath(n int) string {
	return filepath.Join(e.journalsDir, time.Now().AddDate(0, 0, -n).Format("2006_01_02")+".md")
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q): %v", path, err)
	}
	return string(b)
}

func TestCLITodayAppend(t *testing.T) {
	e := newCLIEnv(t)
	_, stderr, err := e.run("t", "--dir", e.logseqDir, "Hello world")
	if err != nil {
		t.Fatalf("unexpected error: %v\nstderr: %s", err, stderr)
	}
	content := readFile(t, e.todayJournalPath())
	if !strings.Contains(content, "Hello world") {
		t.Errorf("expected journal to contain %q, got:\n%s", "Hello world", content)
	}
}

func TestCLITodayMultipleWords(t *testing.T) {
	e := newCLIEnv(t)
	_, stderr, err := e.run("t", "--dir", e.logseqDir, "one", "two", "three")
	if err != nil {
		t.Fatalf("unexpected error: %v\nstderr: %s", err, stderr)
	}
	if !strings.Contains(readFile(t, e.todayJournalPath()), "one two three") {
		t.Errorf("expected journal to contain joined text")
	}
}

func TestCLIAtTagConverted(t *testing.T) {
	e := newCLIEnv(t)
	_, stderr, err := e.run("t", "--dir", e.logseqDir, "hello", "@work")
	if err != nil {
		t.Fatalf("unexpected error: %v\nstderr: %s", err, stderr)
	}
	content := readFile(t, e.todayJournalPath())
	if !strings.Contains(content, "hello #work") {
		t.Errorf("expected journal to contain converted tag, got:\n%s", content)
	}
}

func TestCLIEmailNotConverted(t *testing.T) {
	e := newCLIEnv(t)
	_, stderr, err := e.run("t", "--dir", e.logseqDir, "reach", "me", "at", "foo@bar.com")
	if err != nil {
		t.Fatalf("unexpected error: %v\nstderr: %s", err, stderr)
	}
	content := readFile(t, e.todayJournalPath())
	if strings.Contains(content, "foo#bar.com") {
		t.Errorf("expected email to remain unchanged, got:\n%s", content)
	}
	if !strings.Contains(content, "foo@bar.com") {
		t.Errorf("expected journal to contain email, got:\n%s", content)
	}
}

func TestCLITodayAliases(t *testing.T) {
	for _, cmd := range []string{"today", "t"} {
		e := newCLIEnv(t)
		_, stderr, err := e.run(cmd, "--dir", e.logseqDir, "alias test")
		if err != nil {
			t.Fatalf("%s: unexpected error: %v\nstderr: %s", cmd, err, stderr)
		}
		if !strings.Contains(readFile(t, e.todayJournalPath()), "alias test") {
			t.Errorf("%s: expected journal to contain text", cmd)
		}
	}
}

func TestCLIYesterdayAppend(t *testing.T) {
	e := newCLIEnv(t)
	_, stderr, err := e.run("y", "--dir", e.logseqDir, "Yesterday entry")
	if err != nil {
		t.Fatalf("unexpected error: %v\nstderr: %s", err, stderr)
	}
	if !strings.Contains(readFile(t, e.nDaysAgoJournalPath(1)), "Yesterday entry") {
		t.Errorf("expected yesterday's journal to contain text")
	}
}

func TestCLIYesterdayAliases(t *testing.T) {
	for _, cmd := range []string{"yesterday", "y"} {
		e := newCLIEnv(t)
		_, stderr, err := e.run(cmd, "--dir", e.logseqDir, "alias test")
		if err != nil {
			t.Fatalf("%s: unexpected error: %v\nstderr: %s", cmd, err, stderr)
		}
		if _, statErr := os.Stat(e.nDaysAgoJournalPath(1)); statErr != nil {
			t.Errorf("%s: yesterday journal not created", cmd)
		}
	}
}

func TestCLIAgoAppend(t *testing.T) {
	e := newCLIEnv(t)
	_, stderr, err := e.run("a", "--dir", e.logseqDir, "2", "Past entry")
	if err != nil {
		t.Fatalf("unexpected error: %v\nstderr: %s", err, stderr)
	}
	if !strings.Contains(readFile(t, e.nDaysAgoJournalPath(2)), "Past entry") {
		t.Errorf("expected 2-days-ago journal to contain text")
	}
}

func TestCLIAgoAliases(t *testing.T) {
	for _, cmd := range []string{"ago", "a"} {
		e := newCLIEnv(t)
		_, stderr, err := e.run(cmd, "--dir", e.logseqDir, "3", "entry")
		if err != nil {
			t.Fatalf("%s: unexpected error: %v\nstderr: %s", cmd, err, stderr)
		}
		if _, statErr := os.Stat(e.nDaysAgoJournalPath(3)); statErr != nil {
			t.Errorf("%s: 3-days-ago journal not created", cmd)
		}
	}
}

func TestCLIAgoInvalidN(t *testing.T) {
	e := newCLIEnv(t)
	_, _, err := e.run("a", "--dir", e.logseqDir, "notanumber")
	if err == nil {
		t.Fatal("expected error for non-integer <n>")
	}
}

func TestCLITodayIndent(t *testing.T) {
	e := newCLIEnv(t)
	_, stderr, err := e.run("t", "--dir", e.logseqDir, "--indent=1", "indented entry")
	if err != nil {
		t.Fatalf("unexpected error: %v\nstderr: %s", err, stderr)
	}
	if !strings.Contains(readFile(t, e.todayJournalPath()), "\t- indented entry") {
		t.Errorf("expected indented bullet in journal")
	}
}

func TestCLIShowToday(t *testing.T) {
	e := newCLIEnv(t)
	if err := os.WriteFile(e.todayJournalPath(), []byte("- seeded entry\n"), 0644); err != nil {
		t.Fatal(err)
	}
	stdout, stderr, err := e.run("g", "--dir", e.logseqDir)
	if err != nil {
		t.Fatalf("unexpected error: %v\nstderr: %s", err, stderr)
	}
	if !strings.Contains(stdout, "seeded entry") {
		t.Errorf("expected stdout to contain seeded entry, got:\n%s", stdout)
	}
}

func TestCLIShowAliases(t *testing.T) {
	for _, cmd := range []string{"get", "g"} {
		e := newCLIEnv(t)
		if err := os.WriteFile(e.todayJournalPath(), []byte("- content\n"), 0644); err != nil {
			t.Fatal(err)
		}
		out, _, err := e.run(cmd, "--dir", e.logseqDir)
		if err != nil {
			t.Fatalf("%s error: %v", cmd, err)
		}
		if !strings.Contains(out, "content") {
			t.Errorf("%s: expected content in stdout", cmd)
		}
	}
}

func TestCLIShowYesterday(t *testing.T) {
	e := newCLIEnv(t)
	if err := os.WriteFile(e.nDaysAgoJournalPath(1), []byte("- yesterday\n"), 0644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"g", "--dir", e.logseqDir, "yesterday"},
		{"g", "--dir", e.logseqDir, "y"},
	} {
		out, _, err := e.run(args...)
		if err != nil {
			t.Fatalf("%v error: %v", args, err)
		}
		if !strings.Contains(out, "yesterday") {
			t.Errorf("%v: expected 'yesterday' in stdout", args)
		}
	}
}

func TestCLIShowAgo(t *testing.T) {
	e := newCLIEnv(t)
	if err := os.WriteFile(e.nDaysAgoJournalPath(3), []byte("- three days ago\n"), 0644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"g", "--dir", e.logseqDir, "ago", "3"},
		{"g", "--dir", e.logseqDir, "a", "3"},
	} {
		out, _, err := e.run(args...)
		if err != nil {
			t.Fatalf("%v error: %v", args, err)
		}
		if !strings.Contains(out, "three days ago") {
			t.Errorf("%v: expected content in stdout", args)
		}
	}
}

func TestCLIShowDate(t *testing.T) {
	e := newCLIEnv(t)
	targetPath := filepath.Join(e.journalsDir, "2024_01_15.md")
	if err := os.WriteFile(targetPath, []byte("- dated entry\n"), 0644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"g", "--dir", e.logseqDir, "date", "2024-01-15"},
		{"g", "--dir", e.logseqDir, "d", "2024-01-15"},
	} {
		out, _, err := e.run(args...)
		if err != nil {
			t.Fatalf("%v error: %v", args, err)
		}
		if !strings.Contains(out, "dated entry") {
			t.Errorf("%v: expected content in stdout", args)
		}
	}
}

func TestCLIShowPage(t *testing.T) {
	e := newCLIEnv(t)
	pagePath := filepath.Join(e.pagesDir, "notes.md")
	if err := os.WriteFile(pagePath, []byte("- page content\n"), 0644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"g", "--dir", e.logseqDir, "page", "notes"},
		{"g", "--dir", e.logseqDir, "p", "notes"},
	} {
		out, _, err := e.run(args...)
		if err != nil {
			t.Fatalf("%v error: %v", args, err)
		}
		if !strings.Contains(out, "page content") {
			t.Errorf("%v: expected content in stdout", args)
		}
	}
}

func TestCLIPageOpenAppend(t *testing.T) {
	e := newCLIEnv(t)
	pagePath := filepath.Join(e.pagesDir, "my-page.md")
	if err := os.WriteFile(pagePath, []byte("- existing\n"), 0644); err != nil {
		t.Fatal(err)
	}
	for _, cmd := range []string{"page", "p"} {
		e2 := newCLIEnv(t)
		if err := os.WriteFile(filepath.Join(e2.pagesDir, "my-page.md"), []byte("- existing\n"), 0644); err != nil {
			t.Fatal(err)
		}
		_, stderr, err := e2.run(cmd, "--dir", e2.logseqDir, "my-page", "new entry")
		if err != nil {
			t.Fatalf("%s: unexpected error: %v\nstderr: %s", cmd, err, stderr)
		}
		content := readFile(t, filepath.Join(e2.pagesDir, "my-page.md"))
		if !strings.Contains(content, "new entry") {
			t.Errorf("%s: expected page to contain 'new entry'", cmd)
		}
	}
	_ = pagePath
}

func TestCLIPageAutoExtension(t *testing.T) {
	e := newCLIEnv(t)
	pagePath := filepath.Join(e.pagesDir, "notes.md")
	if err := os.WriteFile(pagePath, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}
	_, stderr, err := e.run("p", "--dir", e.logseqDir, "notes", "appended")
	if err != nil {
		t.Fatalf("unexpected error: %v\nstderr: %s", err, stderr)
	}
	if !strings.Contains(readFile(t, pagePath), "appended") {
		t.Errorf("expected page to contain 'appended'")
	}
}

func TestCLISearchPlainText(t *testing.T) {
	e := newCLIEnv(t)
	if err := os.WriteFile(e.todayJournalPath(), []byte("- TODO buy milk\n- regular note\n"), 0644); err != nil {
		t.Fatal(err)
	}
	for _, cmd := range []string{"search", "s"} {
		stdout, stderr, err := e.run(cmd, "--dir", e.logseqDir, "TODO")
		if err != nil {
			t.Fatalf("%s: unexpected error: %v\nstderr: %s", cmd, err, stderr)
		}
		if !strings.Contains(stdout, "TODO buy milk") {
			t.Errorf("%s: expected TODO line in results", cmd)
		}
		if strings.Contains(stdout, "regular note") {
			t.Errorf("%s: expected regular note excluded", cmd)
		}
	}
}

func TestCLISearchRegex(t *testing.T) {
	e := newCLIEnv(t)
	if err := os.WriteFile(e.todayJournalPath(), []byte("- TODO fix this\n- FIXME that\n- note\n"), 0644); err != nil {
		t.Fatal(err)
	}
	stdout, stderr, err := e.run("search", "--dir", e.logseqDir, "/TODO|FIXME/")
	if err != nil {
		t.Fatalf("unexpected error: %v\nstderr: %s", err, stderr)
	}
	if !strings.Contains(stdout, "TODO fix this") || !strings.Contains(stdout, "FIXME that") {
		t.Errorf("expected both TODO and FIXME lines, got:\n%s", stdout)
	}
	if strings.Contains(stdout, "note\n") {
		t.Errorf("expected plain note excluded, got:\n%s", stdout)
	}
}

func TestCLIFindPrefix(t *testing.T) {
	e := newCLIEnv(t)
	for _, name := range []string{"golang-tips.md", "go-notes.md", "python.md"} {
		if err := os.WriteFile(filepath.Join(e.pagesDir, name), []byte(""), 0644); err != nil {
			t.Fatal(err)
		}
	}
	for _, cmd := range []string{"find", "f"} {
		stdout, stderr, err := e.run(cmd, "--dir", e.logseqDir, "go")
		if err != nil {
			t.Fatalf("%s: unexpected error: %v\nstderr: %s", cmd, err, stderr)
		}
		if !strings.Contains(stdout, "golang-tips.md") && !strings.Contains(stdout, "go-notes.md") {
			t.Errorf("%s: expected go* pages in results, got:\n%s", cmd, stdout)
		}
		if strings.Contains(stdout, "python.md") {
			t.Errorf("%s: expected python.md excluded, got:\n%s", cmd, stdout)
		}
	}
}

func TestCLIDirFlag(t *testing.T) {
	e := newCLIEnv(t)
	_, stderr, err := e.run("t", "--dir="+e.logseqDir, "dir flag test")
	if err != nil {
		t.Fatalf("unexpected error: %v\nstderr: %s", err, stderr)
	}
	if !strings.Contains(readFile(t, e.todayJournalPath()), "dir flag test") {
		t.Errorf("expected journal to contain text")
	}
}

func TestCLIGetJournalRaw(t *testing.T) {
	e := newCLIEnv(t)

	// Write two journal files with distinct content.
	today := e.todayJournalPath()
	yesterday := e.nDaysAgoJournalPath(1)

	writeFile(t, today, "- Today's entry\n- [[MyPage]] note\n- #todo item\n")
	writeFile(t, yesterday, "- Yesterday's entry\n")

	for _, cmd := range [][]string{{"g", "j", "--raw"}, {"get", "journal", "--raw"}} {
		stdout, stderr, err := e.run(cmd...)
		if err != nil {
			t.Fatalf("%v: unexpected error: %v\nstderr: %s", cmd, err, stderr)
		}

		// Raw output should contain the literal markdown text.
		if !strings.Contains(stdout, "Today's entry") {
			t.Errorf("%v: expected today's entry in output, got:\n%s", cmd, stdout)
		}
		if !strings.Contains(stdout, "Yesterday's entry") {
			t.Errorf("%v: expected yesterday's entry in output, got:\n%s", cmd, stdout)
		}

		// Date headers should appear (today before yesterday).
		todayLabel := time.Now().Format("2006-01-02")
		yesterdayLabel := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
		if !strings.Contains(stdout, todayLabel) {
			t.Errorf("%v: expected today date header %q in output, got:\n%s", cmd, todayLabel, stdout)
		}
		if !strings.Contains(stdout, yesterdayLabel) {
			t.Errorf("%v: expected yesterday date header %q in output, got:\n%s", cmd, yesterdayLabel, stdout)
		}

		// Today must appear before yesterday.
		todayIdx := strings.Index(stdout, todayLabel)
		yesterdayIdx := strings.Index(stdout, yesterdayLabel)
		if todayIdx > yesterdayIdx {
			t.Errorf("%v: expected today before yesterday in output", cmd)
		}
	}
}

func TestCLIGetJournalSkipsEmpty(t *testing.T) {
	e := newCLIEnv(t)

	// Write one empty and one non-empty journal.
	today := e.todayJournalPath()
	yesterday := e.nDaysAgoJournalPath(1)

	writeFile(t, today, "")
	writeFile(t, yesterday, "- Only yesterday has content\n")

	stdout, stderr, err := e.run("g", "j", "--raw")
	if err != nil {
		t.Fatalf("unexpected error: %v\nstderr: %s", err, stderr)
	}

	if strings.Contains(stdout, time.Now().Format("2006-01-02")+" ") {
		t.Errorf("expected empty today journal to be skipped, but date header found in output:\n%s", stdout)
	}
	if !strings.Contains(stdout, "Only yesterday has content") {
		t.Errorf("expected yesterday's content in output, got:\n%s", stdout)
	}
}

func TestCLIGetJournalNoEntries(t *testing.T) {
	e := newCLIEnv(t)

	_, stderr, err := e.run("g", "j", "--raw")
	if err != nil {
		t.Fatalf("unexpected error: %v\nstderr: %s", err, stderr)
	}
	if !strings.Contains(stderr, "No journal entries found") {
		t.Errorf("expected 'No journal entries found' message, got stderr: %q", stderr)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile(%q): %v", path, err)
	}
}
