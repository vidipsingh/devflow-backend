package commands

import (
    "fmt"

    "devflow-backend/internal/cli/client"
    "devflow-backend/internal/cli/output"

    "github.com/spf13/cobra"
)

var RepoCmd = &cobra.Command{
	Use: "repo",
	Short: "Manage repositories",
}

var repoListCmd = &cobra.Command{
	Use: "list",
	Short: "List your repositories",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := client.New(true)
		if err != nil { 
			return err 
		}

		data, err := c.Get("/repositories")
		if err != nil { 
			return err 
		}
		m, _ := data.(map[string]any)
		repos, _ := m["repositories"].([]any)
		if len(repos) == 0 {
			output.Info("No repositories found")
			return nil
		}
		rows := make([][]string, 0, len(repos))
		for _, r := range repos {
            repo, _ := r.(map[string]any)
            name, _ := repo["name"].(string)
            vis, _ := repo["visibility"].(string)
            lang, _ := repo["language"].(string)
            desc, _ := repo["description"].(string)
            if len(desc) > 40 { desc = desc[:37] + "..." }
            rows = append(rows, []string{name, vis, lang, desc})
        }
		output.Table([]string{"NAME", "VISIBILITY", "LANGUAGE", "DESCRIPTION"}, rows)
		return nil
	},
}

var repoCreateCmd = &cobra.Command{
	Use: "create <name>",
	Short: "Create a new repository",
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		vis, _ := cmd.Flags().GetString("visibility")
		desc, _ := cmd.Flags().GetString("description")
		c, err := client.New(true)
		if err != nil {
			return err
		}
		_, err = c.Post("/repositories", map[string]any{
            "name":        args[0],
            "visibility":  vis,
            "description": desc,
        })
		if err != nil {
			return err
		}
		output.Success(fmt.Sprintf("Repositories '%s' created", args[0]))
		return nil
	},
}

var repoViewCmd = &cobra.Command{
	Use: "view <name>",
	Short: "Show repositories details",
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := client.New(true)
		if err != nil {
			return err
		}
		data, err := c.Get("/repositories/" + args[0])
		if err != nil {
			return err
		}
		repo, _ := data.(map[string]any)
        stats, _ := repo["stats"].(map[string]any)
        output.Bold(fmt.Sprintf("%v", repo["name"]))
        output.Info(fmt.Sprintf("Visibility  : %v", repo["visibility"]))
        output.Info(fmt.Sprintf("Description : %v", repo["description"]))
        output.Info(fmt.Sprintf("Language    : %v", repo["language"]))
        output.Info(fmt.Sprintf("Stars       : %v", stats["stars"]))
        output.Info(fmt.Sprintf("Open Issues : %v", stats["openIssues"]))
        output.Info(fmt.Sprintf("Default Branch : %v", repo["defaultBranch"]))
        return nil
	},
}

var repoDeleteCmd = &cobra.Command{
	Use: "delete <name>",
	Short: "Delete a repository",
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := client.New(true)
		if err != nil {
			return err
		}
		if err := c.Delete("/repositories/" + args[0]); err != nil {
			return err
		}
		output.Success(fmt.Sprintf("Repository '%s' deleted", args[0]))
		return nil
	},
}

func init() {
    repoCreateCmd.Flags().StringP("visibility", "v", "private", "public or private")
    repoCreateCmd.Flags().StringP("description", "d", "", "Repository description")
    RepoCmd.AddCommand(repoListCmd, repoCreateCmd, repoViewCmd, repoDeleteCmd)
}
