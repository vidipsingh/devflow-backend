package commands

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"devflow-backend/internal/cli/client"
	"devflow-backend/internal/cli/output"
	"devflow-backend/internal/cli/state"

	"github.com/spf13/cobra"
)

var PrCmd = &cobra.Command{
	Use:   "pr",
	Short: "Manage pull requests",
}

var prListCmd = &cobra.Command{
	Use:   "list [repo-name]",
	Short: "List pull requests",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		repo := repoArg(args)
		if repo == "" { return fmt.Errorf("repo name required (or run inside a devflow workspace)") }
		st, _ := cmd.Flags().GetString("state")
		c, err := client.New(true)
		if err != nil { return err }
		data, err := c.Get("/repositories/" + repo + "/pulls?state=" + st)
		if err != nil { return err }
		m, _ := data.(map[string]any)
		prs, _ := m["pullRequests"].([]any)
		if len(prs) == 0 {
			output.Info("No pull requests found"); return nil
		}
		rows := [][]string{}
		for _, p := range prs {
			pr, _ := p.(map[string]any)
			num := fmt.Sprintf("#%v", pr["number"])
			title, _ := pr["title"].(string)
			if len(title) > 45 { title = title[:42] + "..." }
			head, _ := pr["headBranch"].(string)
			base, _ := pr["baseBranch"].(string)
			author, _ := pr["authorName"].(string)
			state, _ := pr["state"].(string)
			rows = append(rows, []string{num, state, author, head + " → " + base, title})
		}
		output.Table([]string{"#", "STATE", "AUTHOR", "BRANCHES", "TITLE"}, rows)
		return nil
	},
}

var prViewCmd = &cobra.Command{
	Use:   "view [repo-name] <number>",
	Short: "View a pull request",
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		repo, numStr := repoAndNum(args)
		if repo == "" { return fmt.Errorf("repo name required") }
		c, err := client.New(true)
		if err != nil { return err }
		data, err := c.Get("/repositories/" + repo + "/pulls/" + numStr)
		if err != nil { return err }
		pr, _ := data.(map[string]any)
		output.Bold(fmt.Sprintf("#%v %v  [%v]", pr["number"], pr["title"], pr["state"]))
		output.Info(fmt.Sprintf("Author    : %v", pr["authorName"]))
		output.Info(fmt.Sprintf("Branches  : %v → %v", pr["headBranch"], pr["baseBranch"]))
		output.Info(fmt.Sprintf("Changes   : +%v -%v  in %v file(s)", pr["additions"], pr["deletions"], lenOf(pr["changedFiles"])))
		output.Dim("─────────────────────────────────")
		output.Dim(fmt.Sprintf("%v", pr["body"]))
		if files, ok := pr["changedFiles"].([]any); ok && len(files) > 0 {
			output.Dim("\nChanged files:")
			for _, f := range files { output.Dim("  " + fmt.Sprintf("%v", f)) }
		}
		if comments, ok := pr["comments"].([]any); ok && len(comments) > 0 {
			output.Dim(fmt.Sprintf("\n%d comment(s):", len(comments)))
			for _, cmt := range comments {
				c, _ := cmt.(map[string]any)
				output.Info(fmt.Sprintf("  @%v: %v", c["authorName"], c["body"]))
			}
		}
		return nil
	},
}

var prCreateCmd = &cobra.Command{
	Use:   "create [repo-name]",
	Short: "Open a new pull request (interactive)",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		repo := repoArg(args)
		if repo == "" { return fmt.Errorf("repo name required") }

		s, _ := state.Load()
		reader := bufio.NewReader(os.Stdin)

		fmt.Print("Title: ")
		title, _ := reader.ReadString('\n')
		title = strings.TrimSpace(title)

		fmt.Print("Body (optional): ")
		body, _ := reader.ReadString('\n')
		body = strings.TrimSpace(body)

		head := s.Branch
		if head == "" { head = "main" }
		fmt.Printf("Head branch [%s]: ", head)
		headInput, _ := reader.ReadString('\n')
		headInput = strings.TrimSpace(headInput)
		if headInput != "" { head = headInput }

		fmt.Print("Base branch [main]: ")
		base, _ := reader.ReadString('\n')
		base = strings.TrimSpace(base)
		if base == "" { base = "main" }

		c, err := client.New(true)
		if err != nil { return err }
		data, err := c.Post("/repositories/"+repo+"/pulls", map[string]any{
			"title":      title,
			"body":       body,
			"headBranch": head,
			"baseBranch": base,
		})
		if err != nil { return err }
		pr, _ := data.(map[string]any)
		output.Success(fmt.Sprintf("PR #%v opened: %v → %v  — \"%v\"", pr["number"], head, base, title))
		return nil
	},
}

var prMergeCmd = &cobra.Command{
	Use:   "merge [repo-name] <number>",
	Short: "Merge a pull request",
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		repo, numStr := repoAndNum(args)
		if repo == "" { return fmt.Errorf("repo name required") }
		method, _ := cmd.Flags().GetString("method")
		if method == "" { method = "merge" }
		c, err := client.New(true)
		if err != nil { return err }
		_, err = c.Post("/repositories/"+repo+"/pulls/"+numStr+"/merge", map[string]any{
			"mergeMethod": method,
		})
		if err != nil { return err }
		output.Success(fmt.Sprintf("PR #%s merged into base branch", numStr))
		return nil
	},
}

var prCloseCmd = &cobra.Command{
	Use:   "close [repo-name] <number>",
	Short: "Close a pull request without merging",
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		repo, numStr := repoAndNum(args)
		if repo == "" { return fmt.Errorf("repo name required") }
		c, err := client.New(true)
		if err != nil { return err }
		_, err = c.Patch("/repositories/"+repo+"/pulls/"+numStr, map[string]string{
			"state": "closed",
		})
		if err != nil { return err }
		output.Success(fmt.Sprintf("PR #%s closed", numStr))
		return nil
	},
}

// repoArg returns the repo name from args[0] or from state.json
func repoArg(args []string) string {
	if len(args) >= 1 { return args[0] }
	if s, err := state.Load(); err == nil { return s.Repo }
	return ""
}

// repoAndNum: if 2 args → (args[0], args[1]); if 1 arg → (state.Repo, args[0])
func repoAndNum(args []string) (string, string) {
	if len(args) == 2 { return args[0], args[1] }
	if s, err := state.Load(); err == nil && s.Repo != "" { return s.Repo, args[0] }
	return "", args[0]
}

func lenOf(v any) int {
	if s, ok := v.([]any); ok { return len(s) }
	return 0
}

func init() {
	prListCmd.Flags().StringP("state", "s", "open", "Filter: open|closed|merged")
	prMergeCmd.Flags().StringP("method", "m", "merge", "Merge method: merge|squash|rebase")
	PrCmd.AddCommand(prListCmd, prViewCmd, prCreateCmd, prMergeCmd, prCloseCmd)
}
