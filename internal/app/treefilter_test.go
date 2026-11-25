package app

import (
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"

	"see/internal/config"
	"see/locales"
)

func TestFilterSavesOpenFolders(t *testing.T) {
	root := t.TempDir()
	keep := filepath.Join(root, "keep")
	docs := filepath.Join(root, "docs")
	a := &App{root: root, quick: quickOpen{files: []string{"docs/Guide.md", "keep/a.go"}}}
	a.tree.Open.Add(keep)

	a.filter.query = "guide"
	a.syncFilter()
	if !a.filtering() || a.filter.tree == nil {
		t.Fatalf("filter did not build: %+v", a.filter)
	}
	if !a.tree.Open.Has(docs) {
		t.Error("a matching folder did not open")
	}

	a.filter.query = ""
	a.syncFilter()
	if a.filtering() || a.filter.tree != nil {
		t.Errorf("filter did not clear: %+v", a.filter)
	}
	if !a.tree.Open.Has(keep) {
		t.Error("the folders open before filtering did not come back")
	}
	if a.tree.Open.Has(docs) {
		t.Error("a folder the filter opened stayed open")
	}
}

func TestFilterHidesConfiguredNames(t *testing.T) {
	root := t.TempDir()
	a := &App{root: root, cfg: config.Load(""), quick: quickOpen{files: []string{".DS_Store", "docs/.DS_Store", "docs/visible.txt"}}}
	a.filter.query = ".DS_Store"
	a.syncFilter()
	if a.filter.matches != 0 {
		t.Fatalf("hidden files appeared in filter: %v", a.filter.roots)
	}
	a.cfg.HiddenNames = "docs"
	a.filter.query = "visible"
	a.syncFilter()
	if a.filter.matches != 0 || !a.hiddenPath(filepath.Join(root, "docs", "visible.txt")) {
		t.Fatal("configured folder did not hide its contents")
	}
	a.cfg.HiddenNames = ""
	a.syncFilter()
	if a.filter.matches != 1 {
		t.Fatal("clearing hidden names did not restore the file")
	}
}

func TestHiddenNamePatternSettings(t *testing.T) {
	root := writeTree(t, map[string]string{
		"go.mod": "", "go.sum": "", ".DS_Store": "", "xDS_Store": "",
		"build12/file.txt": "", "build1/file.txt": "", "Game.csproj": "", "Game.sln": "",
	})
	path := filepath.Join(t.TempDir(), "config.json")
	cfg := config.Load(path)
	cfg.HiddenNamePattern = `\.csproj$`
	h := newHarnessWithConfig(t, root, cfg)
	if h.treeHas(filepath.Join(root, "Game.csproj")) {
		t.Fatal("existing single-line rule did not hide the project file")
	}
	settings := ui.NewTester(func(c *ui.Context) {
		card(c, func() { h.a.hiddenNamePatternRow(c) })
	}, 620, 100)
	field, ok := settings.AccessByRole(ui.RoleTextField, locales.HiddenNamePattern())
	if !ok {
		t.Fatal("missing hidden name pattern field")
	}
	settings.ClickAt(field.Bounds.X+field.Bounds.W/2, field.Bounds.Y+field.Bounds.H/2)
	settings.Key(ui.Cmd, ui.KeyA)
	settings.Type(`\.csproj$`)
	settings.Key(0, ui.KeyEnter)
	settings.Type(`\.sln$`)
	settings.Key(0, ui.KeyEnter)
	settings.SetClipboard("\n" + `^build[0-9]{2,4}$|^go\.mod$`)
	settings.Key(ui.Cmd, ui.KeyV)
	pattern := "\\.csproj$\n\\.sln$\n\n^build[0-9]{2,4}$|^go\\.mod$"
	h.tt.Frame()
	if got := config.Load(path).HiddenNamePattern; got != pattern {
		t.Fatalf("saved patterns = %q, want %q", got, pattern)
	}
	if h.treeHas(filepath.Join(root, "Game.csproj")) || h.treeHas(filepath.Join(root, "Game.sln")) {
		t.Fatal("the separate extension rules did not both apply")
	}
	if !h.treeHas(filepath.Join(root, "go.sum")) || h.treeHas(filepath.Join(root, "go.mod")) {
		t.Fatal("hiding the nesting parent did not promote its visible child")
	}
	if h.treeHas(filepath.Join(root, ".DS_Store")) || !h.treeHas(filepath.Join(root, "xDS_Store")) {
		t.Fatal("an exact name was treated as a regular expression")
	}
	if h.a.canRevealInProjectNavigator(filepath.Join(root, "build12", "file.txt")) {
		t.Fatal("could reveal a file in a hidden folder")
	}
	h.a.quick.files = []string{"build12/file.txt", "build1/file.txt"}
	h.a.filter.query = "file.txt"
	h.tt.Frame()
	if h.a.filter.matches != 1 || h.a.filter.roots[0] != filepath.Join(root, "build1") {
		t.Fatalf("filtered tree shows %v", h.a.filter.roots)
	}
	settings.Key(ui.Cmd, ui.KeyA)
	settings.Type("\\.tmp$\n\n[")
	settings.Frame()
	h.tt.Frame()
	lineError := locales.HiddenNamePatternError("3")
	if !settings.HasText(lineError) || config.Load(path).HiddenNamePattern != pattern || h.a.filter.matches != 1 {
		t.Fatal("invalid edit did not identify its line and preserve the saved rules")
	}
	settings.Key(ui.Cmd, ui.KeyA)
	settings.Type(`\.sln$`)
	h.tt.Frame()
	if settings.HasText(lineError) || h.a.filter.matches != 2 {
		t.Fatal("correcting the list did not clear the error and restore both filter results")
	}
	project, solution := filepath.Join(root, "Game.csproj"), filepath.Join(root, "Game.sln")
	if !h.a.canRevealInProjectNavigator(project) || h.a.canRevealInProjectNavigator(solution) {
		t.Fatal("correcting the list did not apply the remaining rule")
	}
	settings.Key(ui.Cmd, ui.KeyA)
	settings.Key(0, ui.KeyBackspace)
	h.tt.Frame()
	if config.Load(path).HiddenNamePattern != "" || !h.a.canRevealInProjectNavigator(solution) {
		t.Fatal("clearing the field did not remove the last rule")
	}
}

func TestHiddenNamePatternsAfterReload(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	cfg := config.Load(path)
	cfg.HiddenNamePattern = "(?i)\\.csproj$\r\n\r\n^cache[0-9]{2,4}$\r\n^has space $\r\n"
	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}
	a := &App{root: t.TempDir(), cfg: config.Load(path)}
	for name, hidden := range map[string]bool{
		"src/Game.CSPROJ": true, "cache12/main.go": true, "has space ": true,
		"Game.csproj.bak": false, "CACHE12/main.go": false, "has space": false,
	} {
		if got := a.hiddenPath(name); got != hidden {
			t.Errorf("hiddenPath(%q) = %v, want %v", name, got, hidden)
		}
	}
}
