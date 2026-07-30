package commands

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"devflow-backend/internal/cli/client"
	"devflow-backend/internal/cli/config"
	"devflow-backend/internal/cli/output"

	"github.com/spf13/cobra"
)

// extractDateString handles createdAt which can be a plain RFC3339 string
// ("2024-01-15T10:30:00Z") or a BSON extended JSON date object ({"$date":"..."}).
func extractDateString(v any) string {
	switch val := v.(type) {
	case string:
		if len(val) >= 10 {
			return val[:10]
		}
		return val
	case map[string]any:
		// BSON extended JSON: {"$date": "2024-01-15T10:30:00Z"}
		if d, ok := val["$date"].(string); ok && len(d) >= 10 {
			return d[:10]
		}
	}
	return ""
}



// extractRepoName parses a repo name from:
//   - plain name:             "my-repo"
//   - frontend URL:           "http://localhost:3000/dashboard/repositories/my-repo"
//   - backend/API URL:        "http://localhost:8080/user/my-repo"
//   - any trailing URL form:  "https://devflow.io/user/my-repo"
func extractRepoName(arg string) string {
	arg = strings.TrimRight(arg, "/")
	if !strings.Contains(arg, "/") {
		return arg
	}
	// If it contains /repositories/ (frontend URL pattern), extract what comes after
	if idx := strings.LastIndex(arg, "/repositories/"); idx != -1 {
		rest := arg[idx+len("/repositories/"):]
		// strip any trailing path segments (e.g. /issues)
		if slash := strings.Index(rest, "/"); slash != -1 {
			rest = rest[:slash]
		}
		return rest
	}
	// Fallback: last path segment
	parts := strings.Split(arg, "/")
	return parts[len(parts)-1]
}

// clone
var CloneCmd = &cobra.Command{
	Use:   "clone <repo-name-or-url>",
	Short: "Download all files from a DevFlow repo to a local folder",
	Long: `Download all files from a DevFlow repository.

Accepts a plain name or a URL from the DevFlow web UI:
  devflow clone my-repo
  devflow clone http://localhost:3000/dashboard/repositories/my-repo`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := extractRepoName(args[0])
		c, err := client.New(true)
		if err != nil {
			return err
		}

		// 1. Get file tree
		data, err := c.Get("/repositories/" + name + "/tree")
		if err != nil {
			return err
		}
		m, _ := data.(map[string]any)
		entries, _ := m["tree"].([]any)

		cfg, _ := config.Load()
		dest := filepath.Join(".", name)
		if err := os.MkdirAll(dest, 0755); err != nil {
			return err
		}

		for _, e := range entries {
			entry, _ := e.(map[string]any)
			if entry["type"] == "dir" {
				continue
			}
			filePath, _ := entry["path"].(string)

			// 2. Fetch file content
			url := cfg.Host + "/api/v1/repositories/" + name + "/blob?path=" + filePath
			req, _ := http.NewRequest("GET", url, nil)
			req.Header.Set("Authorization", "Bearer "+cfg.Token)
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				output.Warn(fmt.Sprintf("skipping %s: %v", filePath, err))
				continue
			}
			localPath := filepath.Join(dest, filepath.FromSlash(filePath))
			os.MkdirAll(filepath.Dir(localPath), 0755)
			f, err := os.Create(localPath)
			if err != nil {
				resp.Body.Close()
				continue
			}
			io.Copy(f, resp.Body)
			f.Close()
			resp.Body.Close()
			output.Info("  " + filePath)
		}
		output.Success(fmt.Sprintf("Cloned '%s' → ./%s/", name, name))
		return nil
	},
}

// push
var skipDirs = map[string]bool{
	".git": true, "node_modules": true, ".next": true,
	"vendor": true, "__pycache__": true, ".DS_Store": true,
}

var PushCmd = &cobra.Command{
	Use:   "push <repo-name>",
	Short: "Upload all local files to a DevFlow repo",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		c, err := client.New(true)
		if err != nil {
			return err
		}

		// Walk current directory
		count := 0
		err = filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			parts := strings.Split(filepath.ToSlash(path), "/")
			for _, part := range parts {
				if skipDirs[part] {
					return filepath.SkipDir
				}
			}
			if info.IsDir() {
				return nil
			}

			repoPath := filepath.ToSlash(path)
			if err := c.PostMultipart("/repositories/"+name+"/files", path, repoPath); err != nil {
				output.Warn(fmt.Sprintf("failed %s: %v", repoPath, err))
			} else {
				output.Info("  pushed: " + repoPath)
				count++
			}
			return nil
		})
		if err != nil {
			return err
		}
		output.Success(fmt.Sprintf("Pushed %d files to '%s'", count, name))
		return nil
	},
}

// ls
var LsCmd = &cobra.Command{
	Use:   "ls <repo-name> [path]",
	Short: "List files in a repository",
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		path := ""
		if len(args) == 2 {
			path = "?path=" + args[1]
		}
		c, err := client.New(true)
		if err != nil {
			return err
		}
		data, err := c.Get("/repositories/" + name + "/tree" + path)
		if err != nil {
			return err
		}
		m, _ := data.(map[string]any)
		entries, _ := m["tree"].([]any)
		for _, e := range entries {
			entry, _ := e.(map[string]any)
			t, _ := entry["type"].(string)
			p, _ := entry["path"].(string)
			if t == "dir" {
				output.Info("📁 " + p + "/")
			} else {
				output.Dim("   " + p)
			}
		}
		return nil
	},
}

// cat
var CatCmd = &cobra.Command{
	Use:   "cat <repo-name> <file-path>",
	Short: "Print a file from a repository",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		name, filePath := args[0], args[1]
		cfg, _ := config.Load()
		url := cfg.Host + "/api/v1/repositories/" + name + "/blob?path=" + filePath
		req, _ := http.NewRequest("GET", url, nil)
		req.Header.Set("Authorization", "Bearer "+cfg.Token)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		io.Copy(os.Stdout, resp.Body)
		return nil
	},
}

// commits
var CommitsCmd = &cobra.Command{
	Use:   "commits <repo-name>",
	Short: "Show commit history",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := client.New(true)
		if err != nil {
			return err
		}
		data, err := c.Get("/repositories/" + args[0] + "/commits")
		if err != nil {
			return err
		}
		m, _ := data.(map[string]any)
		commits, _ := m["commits"].([]any)
		if len(commits) == 0 {
			output.Info("No commits yet")
			return nil
		}
		rows := [][]string{}
		for _, commit := range commits {
			cmt, _ := commit.(map[string]any)
			sha, _ := cmt["shortHash"].(string)
			msg, _ := cmt["message"].(string)
			if len(msg) > 60 {
				msg = msg[:57] + "..."
			}
			author, _ := cmt["authorName"].(string)
			date := extractDateString(cmt["createdAt"])
			rows = append(rows, []string{sha, date, author, msg})
		}
		output.Table([]string{"SHA", "DATE", "AUTHOR", "MESSAGE"}, rows)
		return nil
	},
}
