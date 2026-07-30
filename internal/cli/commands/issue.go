package commands

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"devflow-backend/internal/cli/client"
	"devflow-backend/internal/cli/output"

	"github.com/spf13/cobra"
)

var IssueCmd = &cobra.Command{
	Use:   "issue",
	Short: "Manage issues",
}

var issueListCmd = &cobra.Command{
	Use:   "list <repo-name>",
	Short: "List open issues",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := client.New(true)
		if err != nil {
			return err
		}
		status, _ := cmd.Flags().GetString("status")
		data, err := c.Get("/repositories/" + args[0] + "/issues?status=" + status)
		if err != nil {
			return err
		}
		m, _ := data.(map[string]any)
		issues, _ := m["issues"].([]any)
		if len(issues) == 0 {
			output.Info("No issues found")
			return nil
		}
		rows := [][]string{}
		for _, i := range issues {
			issue, _ := i.(map[string]any)
			num := fmt.Sprintf("#%v", issue["number"])
			title, _ := issue["title"].(string)
			if len(title) > 50 {
				title = title[:47] + "..."
			}
			author, _ := issue["authorName"].(string)
			st, _ := issue["state"].(string)
			rows = append(rows, []string{num, st, author, title})
		}
		output.Table([]string{"#", "STATUS", "AUTHOR", "TITLE"}, rows)
		return nil
	},
}

var issueViewCmd = &cobra.Command{
	Use:   "view <repo-name> <number>",
	Short: "View a specific issue",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := client.New(true)
		if err != nil {
			return err
		}
		data, err := c.Get("/repositories/" + args[0] + "/issues/" + args[1])
		if err != nil {
			return err
		}
		issue, _ := data.(map[string]any)
		output.Bold(fmt.Sprintf("#%v %v", issue["number"], issue["title"]))
		output.Info(fmt.Sprintf("Status  : %v", issue["state"]))
		output.Info(fmt.Sprintf("Author  : %v", issue["authorName"]))
		output.Dim("───────────────────────────────────────")
		output.Dim(fmt.Sprintf("%v", issue["body"]))
		comments, _ := issue["comments"].([]any)
		if len(comments) > 0 {
			output.Dim(fmt.Sprintf("\n%d comments", len(comments)))
			for _, cmt := range comments {
				c, _ := cmt.(map[string]any)
				output.Info(fmt.Sprintf("  @%v: %v", c["authorName"], c["body"]))
			}
		}
		return nil
	},
}

var issueCreateCmd = &cobra.Command{
	Use:   "create <repo-name>",
	Short: "Create a new issue (interactive)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Title: ")
		title, _ := reader.ReadString('\n')
		title = strings.TrimSpace(title)
		fmt.Print("Body (single line): ")
		body, _ := reader.ReadString('\n')
		body = strings.TrimSpace(body)

		c, err := client.New(true)
		if err != nil {
			return err
		}
		_, err = c.Post("/repositories/"+args[0]+"/issues", map[string]any{
			"title": title,
			"body":  body,
		})
		if err != nil {
			return err
		}
		output.Success(fmt.Sprintf("Issue created in '%s'", args[0]))
		return nil
	},
}

var issueCloseCmd = &cobra.Command{
	Use:   "close <repo-name> <number>",
	Short: "Close an issue",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := client.New(true)
		if err != nil {
			return err
		}
		_, err = c.Patch("/repositories/"+args[0]+"/issues/"+args[1], map[string]string{
			"state": "closed",
		})
		if err != nil {
			return err
		}
		output.Success(fmt.Sprintf("Issue #%s closed", args[1]))
		return nil
	},
}

func init() {
	issueListCmd.Flags().StringP("status", "s", "open", "Filter by status: open or closed")
	IssueCmd.AddCommand(issueListCmd, issueViewCmd, issueCreateCmd, issueCloseCmd)
}
