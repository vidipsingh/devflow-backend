package main

import (
    "fmt"
    "os"

    "devflow-backend/internal/cli/commands"
    "devflow-backend/internal/cli/config"

    "github.com/spf13/cobra"
)

func main() {
    root := &cobra.Command{
        Use:   "devflow",
        Short: "DevFlow CLI — manage repos, files, and issues from your terminal",
        Long: `
  ____              _____ _               
 |  _ \  _____   _|  ___| | _____      __
 | | | |/ _ \ \ / / |_  | |/ _ \ \ /\ / /
 | |_| |  __/\ V /|  _| | | (_) \ V  V / 
 |____/ \___| \_/ |_|   |_|\___/ \_/\_/  

 Manage repositories, push/pull files, and track issues.
`,
        SilenceUsage: true,
    }

    // Global --host flag to override the API base URL
    root.PersistentFlags().String("host", "", "Override API host (e.g. https://api.devflow.io)")

    // Register host override before commands run
    cobra.OnInitialize(func() {
        host, _ := root.PersistentFlags().GetString("host")
        if host != "" {
            cfg, _ := config.Load()
            cfg.Host = host
            config.Save(cfg)
        }
    })

    // Register all command groups
    root.AddCommand(
        commands.AuthCmd,
        commands.RepoCmd,
        commands.CloneCmd,
        commands.PushCmd,
        commands.LsCmd,
        commands.CatCmd,
        commands.CommitsCmd,
        commands.IssueCmd,
    )

    if err := root.Execute(); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}
