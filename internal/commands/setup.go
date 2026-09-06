package commands

import (
	"io/fs"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/symbolsecurity/cli/internal/output"
	"github.com/symbolsecurity/cli/skills"
)

func skillDirs(home string) []string {
	return []string{
		filepath.Join(home, ".config", "opencode", "skills"),
		filepath.Join(home, ".claude", "skills"),
		filepath.Join(home, ".codex", "skills"),
		filepath.Join(home, ".cursor", "skills"),
		filepath.Join(home, ".codeium", "windsurf", "skills"),
	}
}

func agentPresent(home, skillsDir string) bool {
	switch {
	case filepath.Base(filepath.Dir(skillsDir)) == "opencode":
		_, err := os.Stat(filepath.Join(home, ".config", "opencode"))
		return err == nil
	case filepath.Base(filepath.Dir(skillsDir)) == ".claude" || filepath.Base(skillsDir) == "skills" && filepath.Base(filepath.Dir(skillsDir)) == ".claude":
		_, err := os.Stat(filepath.Join(home, ".claude"))
		return err == nil
	}
	parent := filepath.Dir(skillsDir)
	_, err := os.Stat(parent)
	return err == nil
}

func (rt *Runtime) setupCmd() *cobra.Command {
	var allAgents bool
	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Install the embedded agent skill into detected agents",
		RunE: func(cmd *cobra.Command, args []string) error {
			written := []map[string]string{}
			for _, dir := range skillDirs(rt.Env.Home) {
				if !allAgents && !agentPresent(rt.Env.Home, dir) {
					continue
				}
				dest := filepath.Join(dir, "symbol")
				if err := os.MkdirAll(dest, 0o755); err != nil {
					return rt.Out.Fail(err)
				}
				if err := copySkill(dest); err != nil {
					return rt.Out.Fail(err)
				}
				written = append(written, map[string]string{"path": dest})
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
