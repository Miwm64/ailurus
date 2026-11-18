// go test ./golden_test            check
// go test ./golden_test -update    rewrite the expected fields from the actual run
package golden_test

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

const module = "github.com/Miwm64/ailurus"

var update = flag.Bool("update", false, "rewrite the expected fields of the golden files")

var buildTool, runTool string

type golden struct {
	Description string            `yaml:"description,omitempty"`
	Sources     map[string]string `yaml:"sources"`
	BuildArgs   []string          `yaml:"build_args,omitempty"`
	Config      string            `yaml:"config,omitempty"` // ailurus-run -c

	Summary string `yaml:"summary"` // ailurus-run stderr
	Output  string `yaml:"output"`  // ailurus-run stdout
	Listing string `yaml:"listing"` // ailurus-build -l
	Journal string `yaml:"journal"` // ailurus-run -j
}

func TestMain(m *testing.M) {
	flag.Parse()

	dir, err := os.MkdirTemp("", "ailurus-golden")
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	buildTool, runTool = filepath.Join(dir, "ailurus-build"), filepath.Join(dir, "ailurus-run")
	for path, pkg := range map[string]string{buildTool: "/cmd/ailurus-build", runTool: "/cmd/ailurus-run"} {
		build := exec.CommandContext(context.Background(), "go", "build", "-o", path, module+pkg)
		if out, err := build.CombinedOutput(); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "build %s: %v\n%s", pkg, err, out)
			_ = os.RemoveAll(dir)
			os.Exit(1)
		}
	}

	code := m.Run()
	_ = os.RemoveAll(dir)
	os.Exit(code)
}

func TestGolden(t *testing.T) {
	t.Parallel()

	files, err := filepath.Glob("testdata/*.yml")
	if err != nil || len(files) == 0 {
		t.Fatalf("no golden files in testdata/: %v", err)
	}

	for _, file := range files {
		t.Run(strings.TrimSuffix(filepath.Base(file), ".yml"), func(t *testing.T) {
			t.Parallel()

			want := load(t, file)
			got := run(t, want)

			if *update {
				save(t, file, got)

				return
			}

			compare(t, "listing", got.Listing, want.Listing)
			compare(t, "summary", got.Summary, want.Summary)
			compare(t, "output", got.Output, want.Output)
			compare(t, "journal", got.Journal, want.Journal)
		})
	}
}

func run(t *testing.T, g golden) golden {
	t.Helper()

	dir := t.TempDir()
	names := make([]string, 0, len(g.Sources))

	for name, text := range g.Sources {
		write(t, dir, name, text)
		names = append(names, name)
	}

	slices.Sort(names)

	args := append([]string{"-o", "prog.bin", "-l", "prog.lst"}, g.BuildArgs...)
	if _, stderr, err := command(t.Context(), dir, buildTool, append(args, names...)...); err != nil {
		t.Fatalf("%v\n%s", err, stderr)
	}

	args = []string{"-l", "prog.lst", "-j", "journal.txt"}
	if g.Config != "" {
		write(t, dir, "config.yml", g.Config)
		args = append(args, "-c", "config.yml")
	}

	var exitErr *exec.ExitError

	stdout, stderr, err := command(t.Context(), dir, runTool, append(args, "prog.bin")...)
	if err != nil && (!errors.As(err, &exitErr) || exitErr.ExitCode() != 1) {
		t.Fatalf("%v\n%s", err, stderr)
	}

	g.Listing = read(t, dir, "prog.lst")
	g.Summary = strings.TrimSpace(stderr)
	g.Output = stdout
	g.Journal = read(t, dir, "journal.txt")

	return g
}

func command(ctx context.Context, dir, tool string, args ...string) (string, string, error) {
	var stdout, stderr bytes.Buffer

	cmd := exec.CommandContext(ctx, tool, args...)
	cmd.Dir, cmd.Stdout, cmd.Stderr = dir, &stdout, &stderr

	if err := cmd.Run(); err != nil {
		return stdout.String(), stderr.String(), fmt.Errorf("%s: %w", filepath.Base(tool), err)
	}

	return stdout.String(), stderr.String(), nil
}

func load(t *testing.T, file string) golden {
	t.Helper()

	f, err := os.Open(file)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()

	var g golden

	dec := yaml.NewDecoder(f)
	dec.KnownFields(true)

	if err := dec.Decode(&g); err != nil {
		t.Fatalf("%s: %v", file, err)
	}

	return g
}

func save(t *testing.T, file string, g golden) {
	t.Helper()

	var buf bytes.Buffer

	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)

	if err := enc.Encode(g); err != nil {
		t.Fatal(err)
	}

	if err := enc.Close(); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(file, buf.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
}

func write(t *testing.T, dir, name, text string) {
	t.Helper()

	if err := os.WriteFile(filepath.Join(dir, name), []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
}

func read(t *testing.T, dir, name string) string {
	t.Helper()

	b, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		t.Fatal(err)
	}

	return string(b)
}

func compare(t *testing.T, field, got, want string) {
	t.Helper()

	if got == want {
		return
	}

	g, w := strings.Split(got, "\n"), strings.Split(want, "\n")
	for i := range max(len(g), len(w)) {
		if gl, wl := lineAt(g, i), lineAt(w, i); gl != wl {
			t.Errorf("%s, line %d:\n got  %s\n want %s\n(go test ./golden_test -update rewrites the file)",
				field, i+1, gl, wl)

			return
		}
	}
}

func lineAt(lines []string, i int) string {
	if i >= len(lines) {
		return "(no line)"
	}

	return strconv.Quote(lines[i])
}
