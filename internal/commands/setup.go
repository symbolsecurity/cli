package commands

import (
	"io/fs"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/symbolsecurity/cli/internal/output"
	"github.com/symbolsecurity/cli/skills"
)

type skillTarget struct {
	Name   string
	Skills string
	Marker string
}

func skillTargets(home string) []skillTarget {
	return []skillTarget{
		{Name: "opencode", Skills: filepath.Join(home, ".config", "opencode", "skills"), Marker: filepath.Join(home, ".config", "opencode")},
		{Name: "claude", Skills: filepath.Join(home, ".claude", "skills"), Marker: filepath.Join(home, ".claude")},
		{Name: "codex", Skills: filepath.Join(home, ".codex", "skills"), Marker: filepath.Join(home, ".codex")},
		{Name: "cursor", Skills: filepath.Join(home, ".cursor", "skills"), Marker: filepath.Join(home, ".cursor")},
		{Name: "windsurf", Skills: filepath.Join(home, ".codeium", "windsurf", "skills"), Marker: filepath.Join(home, ".codeium", "windsurf")},
	}
}

func skillDirs(home string) []string {
	var dirs []string
	for _, t := range skillTargets(home) {
		dirs = append(dirs, t.Skills)
	}
	return dirs
}

func (rt *Runtime) setupCmd() *cobra.Command {
	var allAgents bool
	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Install the embedded agent skill into detected agents",
		RunE: func(cmd *cobra.Command, args []string) error {
			written := []map[string]string{}
			for _, t := range skillTargets(rt.Env.Home) {
				if !allAgents {
					if _, err := os.Stat(t.Marker); err != nil {
						continue
					}
				}
				dest := filepath.Join(t.Skills, "symbol")
				if err := os.MkdirAll(dest, 0o755); err != nil {
					return rt.Out.Fail(err)
				}
				if err := copySkill(dest); err != nil {
					return rt.Out.Fail(err)
				}
				written = append(written, map[string]string{"agent": t.Name, "path": dest})
			}
			if len(written) == 0 {
				return rt.Out.Fail(output.Usage("no agent skill directories detected", "Create an agent config dir or pass --all-agents"))
			}
			return rt.Out.Success(map[string]any{"installed": written}, "skill installed", []output.Breadcrumb{
				crumb("login", "symbol auth login"),
				crumb("doctor", "symbol doctor"),
			}, nil)
		},
	}
	cmd.Flags().BoolVar(&allAgents, "all-agents", false, "Write into all known skill paths")
	return cmd
}

func copySkill(dest string) error {
	return fs.WalkDir(skills.FS, "symbol", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel("symbol", path)
		if err != nil {
			return err
		}
		target := filepath.Join(dest, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		b, err := skills.FS.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, b, 0o644)
	})
}
