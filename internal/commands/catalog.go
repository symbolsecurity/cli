package commands

import (
	"github.com/spf13/cobra"
)

func (rt *Runtime) commandsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "commands",
		Short: "Dump the command catalog",
		RunE: func(cmd *cobra.Command, args []string) error {
			root := cmd.Root()
			var items []map[string]any
			var walk func(c *cobra.Command, path []string)
			walk = func(c *cobra.Command, path []string) {
				if c.Hidden || c.Name() == "help" {
					return
				}
				if len(path) > 0 {
					item := map[string]any{
						"path":  path,
						"use":   c.UseLine(),
						"short": c.Short,
					}
					if c.Annotations != nil && c.Annotations["gotchas"] != "" {
						item["gotchas"] = c.Annotations["gotchas"]
					}
					items = append(items, item)
				}
				for _, ch := range c.Commands() {
					p := append(append([]string{}, path...), ch.Name())
					walk(ch, p)
				}
			}
			walk(root, nil)
			return rt.Out.Success(map[string]any{"commands": items}, plural(len(items), "command"), nil, nil)
		},
	}
}
