package commands

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"devflow-backend/internal/cli/client"
	"devflow-backend/internal/cli/config"
	"devflow-backend/internal/cli/output"
	"devflow-backend/internal/cli/state"

	"github.com/spf13/cobra"
)

// init
var InitCmd = &cobra.Command{
	Use:   "init [repo-name]",
	Short: "Initialise a local DevFlow workspace in the current directory",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s, _ := state.Load()
		// Always resolve repo name: explicit arg > current directory name
		if len(args) == 1 {
			s.Repo = args[0]
		} else {
			// Always default to current directory name — never keep stale state
			cwd, _ := os.Getwd()
			s.Repo = filepath.Base(cwd)
		}
		if s.Branch == "" {
			s.Branch = "main"
		}
		s.Staged = []string{}
		if err := state.Save(s); err != nil {
			return err
		}
		output.Success(fmt.Sprintf("Initialised workspace → repo: %s, branch: %s", s.Repo, s.Branch))
		output.Dim("  .devflow/state.json created")
		return nil
	},
}

// add
var AddCmd = &cobra.Command{
	Use:   "add <file|.>",
	Short: "Stage files for next commit",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if !state.Exists() {
			return fmt.Errorf("not a devflow workspace - run: devflow init")
		}
		var paths []string
		for _, arg := range args {
			if arg == "." {
				filepath.Walk(".", func(p string, info os.FileInfo, err error) error {
					if err != nil || info.IsDir() {
						return nil
					}
					parts := strings.Split(filepath.ToSlash(p), "/")
					for _, part := range parts {
						if skipDirs[part] {
							return filepath.SkipDir
						}
					}
					paths = append(paths, filepath.ToSlash(p))
					return nil
				})
			} else {
				paths = append(paths, filepath.ToSlash(arg))
			}
		}
		if err := state.Stage(paths); err != nil {
			return err
		}
		output.Success(fmt.Sprintf("Staged %d file(s)", len(paths)))
		for _, p := range paths {
			output.Info(" +" + p)
		}
		return nil
	},
}

// status
var StatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show staged and unstaged files",
	RunE: func(cmd *cobra.Command, args []string) error {
		if !state.Exists() {
			return fmt.Errorf("not a devlow workspace - run: devflow init")
		}
		s, err := state.Load()
		if err != nil {
			return err
		}

		output.Bold(fmt.Sprintf("On branch %s  |  repo :%s", s.Branch, s.Repo))
		output.Dim("─────────────────────────────────")

		// collect all local files
		localFiles := map[string]bool{}
		filepath.Walk(".", func(p string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}
			parts := strings.Split(filepath.ToSlash(p), "/")
			for _, part := range parts {
				if skipDirs[part] {
					return filepath.SkipDir
				}
			}
			localFiles[filepath.ToSlash(p)] = true
			return nil
		})

		stagedSet := map[string]bool{}
		for _, p := range s.Staged {
			stagedSet[p] = true
		}

		if len(s.Staged) > 0 {
			fmt.Println("\033[32mStaged (will be committed):\033[0m")
			for _, p := range s.Staged {
				fmt.Println("  \033[32m+ " + p + "\033[0m")
			}
		}

		unstaged := []string{}
		for p := range localFiles {
			if !stagedSet[p] {
				unstaged = append(unstaged, p)
			}
		}
		if len(unstaged) > 0 {
			fmt.Println("\033[90mUnstaged (not included in next commit):\033[0m")
			for _, p := range unstaged {
				fmt.Println("  \033[90m  " + p + "\033[0m")
			}
		}

		if len(s.Staged) == 0 && len(unstaged) == 0 {
			output.Info("Nothing to commit — working directory clean")
		}
		return nil
	},
}

// commit
var CommitCmd = &cobra.Command{
	Use:   "commit",
	Short: "Upload staged files to the server as a new commit",
	RunE: func(cmd *cobra.Command, args []string) error {
		if !state.Exists() {
			return fmt.Errorf("not a devflow workspace - run: devflow init")
		}
		msg, _ := cmd.Flags().GetString("message")
		if msg == "" {
			reader := bufio.NewReader(os.Stdin)
			fmt.Print("Commit message: ")
			msg, _ = reader.ReadString('\n')
			msg = strings.TrimSpace(msg)
		}
		if msg == "" {
			return fmt.Errorf("commit message is required")
		}

		s, err := state.Load()
		if err != nil {
			return err
		}
		if len(s.Staged) == 0 {
			return fmt.Errorf("nothing staged - run: devflow add .")
		}

		c, err := client.New(true)
		if err != nil {
			return err
		}

		ok := 0
		for _, path := range s.Staged {
			raw, err := os.ReadFile(path)
			if err != nil {
				output.Warn(fmt.Sprintf("skip %s: %v", path, err))
				continue
			}
			encoded := base64.StdEncoding.EncodeToString(raw)
			_, err = c.Post("/repositories/"+s.Repo+"/files", map[string]any{
				"path":    path,
				"content": encoded,
				"message": msg,
				"branch":  s.Branch,
			})
			if err != nil {
				output.Warn(fmt.Sprintf("failed %s: %v", path, err))
			} else {
				output.Info("  committed: " + path)
				ok++
			}
		}

		if err := state.ClearStaged(); err != nil {
			return err
		}
		output.Success(fmt.Sprintf("Committed %d files(s) to %s/%s", ok, s.Repo, s.Branch))
		return nil
	},
}

// checkout
var CheckoutCmd = &cobra.Command{
	Use:   "checkout <branch>",
	Short: "Switch branch (use -b to create a new branch)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if !state.Exists() {
			return fmt.Errorf("not a devflow workspace - run: devflow init")
		}
		newBranch := args[0]
		create, _ := cmd.Flags().GetBool("b")

		s, err := state.Load()
		if err != nil {
			return err
		}

		if create {
			// Tell server to create the branch by sending an empty placeholder commit
			c, err := client.New(true)
			if err != nil {
				return err
			}
			// We signal branch creation by uploading a .devflow-branch placeholder
			placeholder := base64.StdEncoding.EncodeToString([]byte("branch: " + newBranch))
			_, _ = c.Post("/repositories/"+s.Repo+"/files", map[string]any{
				"path":    ".devflow-branch",
				"content": placeholder,
				"message": "Create branch " + newBranch,
				"branch":  newBranch,
			})
			output.Success(fmt.Sprintf("Created branch '%s' on server", newBranch))
		}

		s.Branch = newBranch
		if err := state.Save(s); err != nil {
			return err
		}
		output.Success(fmt.Sprintf("Switched to branch '%s'", newBranch))
		return nil
	},
}

// branch
var BranchCmd = &cobra.Command{
	Use:   "branch",
	Short: "List branches in the current repo",
	RunE: func(cmd *cobra.Command, args []string) error {
		s, err := state.Load()
		if err != nil || s.Repo == "" {
			return fmt.Errorf("not in a devflow workspace — run: devflow init")
		}
		c, err := client.New(true)
		if err != nil { return err }
		data, err := c.Get("/repositories/" + s.Repo)
		if err != nil { return err }
		repo, _ := data.(map[string]any)
		branches, _ := repo["branches"].([]any)
		
		output.Bold(fmt.Sprintf("Branches for %s:", s.Repo))
		// Always show current branch even if server list is empty
		found := false
		for _, b := range branches {
			bStr, _ := b.(string)
			if bStr == s.Branch { found = true }
			if bStr == s.Branch {
				fmt.Println("  \033[32m* " + bStr + "\033[0m  ← current")
			} else {
				output.Dim("    " + bStr)
			}
		}
		if !found {
			fmt.Println("  \033[32m* " + s.Branch + "\033[0m  ← current (local only)")
		}
		return nil
	},
}

var PushCmd2 = &cobra.Command{
    Use:   "push",
    Short: "Push committed changes to the remote repository (uploads staged files)",
    RunE: func(cmd *cobra.Command, args []string) error {
        if !state.Exists() {
            return fmt.Errorf("not a devflow workspace — run: devflow init")
        }
        s, err := state.Load()
        if err != nil { return err }
        if len(s.Staged) == 0 {
            output.Info("Nothing to push — working tree clean")
            return nil
        }
        // Re-use commit logic with auto message
        msg, _ := cmd.Flags().GetString("message")
        if msg == "" { msg = "Push from CLI" }
        
        c, err := client.New(true)
        if err != nil { return err }
        ok := 0
        for _, path := range s.Staged {
            raw, err := os.ReadFile(path)
            if err != nil { output.Warn(fmt.Sprintf("skip %s: %v", path, err)); continue }
            encoded := base64.StdEncoding.EncodeToString(raw)
            _, err = c.Post("/repositories/"+s.Repo+"/files", map[string]any{
                "path": path, "content": encoded,
                "message": msg, "branch": s.Branch,
            })
            if err != nil { output.Warn(fmt.Sprintf("failed %s: %v", path, err)) } else {
                output.Info("  pushed: " + path); ok++
            }
        }
        state.ClearStaged()
        output.Success(fmt.Sprintf("Pushed %d file(s) to %s/%s", ok, s.Repo, s.Branch))
        return nil
    },
}

var PullCmd = &cobra.Command{
    Use:   "pull",
    Short: "Pull latest files from the remote branch into the current directory",
    RunE: func(cmd *cobra.Command, args []string) error {
        if !state.Exists() {
            return fmt.Errorf("not a devflow workspace — run: devflow init")
        }
        s, err := state.Load()
        if err != nil { return err }
        
        cfg, _ := config.Load()
        c, err := client.New(true)
        if err != nil { return err }
        
        // Get file tree for current branch
        data, err := c.Get("/repositories/" + s.Repo + "/tree?ref=" + s.Branch)
        if err != nil { return err }
        m, _ := data.(map[string]any)
        entries, _ := m["tree"].([]any)
        
        count := 0
        for _, e := range entries {
            entry, _ := e.(map[string]any)
            if entry["type"] == "dir" { continue }
            filePath, _ := entry["path"].(string)
            
            url := cfg.Host + "/api/v1/repositories/" + s.Repo + "/blob?path=" + filePath + "&ref=" + s.Branch
            req, _ := http.NewRequest("GET", url, nil)
            req.Header.Set("Authorization", "Bearer "+cfg.Token)
            resp, err := http.DefaultClient.Do(req)
            if err != nil { output.Warn("skip " + filePath); continue }
            
            os.MkdirAll(filepath.Dir(filePath), 0755)
            f, err := os.Create(filePath)
            if err != nil { resp.Body.Close(); continue }
            io.Copy(f, resp.Body)
            f.Close()
            resp.Body.Close()
            output.Info("  pulled: " + filePath)
            count++
        }
        output.Success(fmt.Sprintf("Pulled %d file(s) from %s/%s", count, s.Repo, s.Branch))
        return nil
    },
}

var LogCmd = &cobra.Command{
    Use:   "log",
    Short: "Show commit history for current repo and branch",
    RunE: func(cmd *cobra.Command, args []string) error {
        s, err := state.Load()
        if err != nil || s.Repo == "" {
            return fmt.Errorf("not in a devflow workspace — run: devflow init")
        }
        c, err := client.New(true)
        if err != nil { return err }
        data, err := c.Get("/repositories/" + s.Repo + "/commits?ref=" + s.Branch)
        if err != nil { return err }
        m, _ := data.(map[string]any)
        commits, _ := m["commits"].([]any)
        if len(commits) == 0 {
            output.Info("No commits yet"); return nil
        }
        for _, commit := range commits {
            cmt, _ := commit.(map[string]any)
            sha, _ := cmt["shortHash"].(string)
            msg, _ := cmt["message"].(string)
            author, _ := cmt["authorName"].(string)
            date := extractDateString(cmt["createdAt"])
            fmt.Printf("\033[33mcommit %s\033[0m\n", sha)
            fmt.Printf("Author: %s\n", author)
            fmt.Printf("Date:   %s\n", date)
            fmt.Printf("\n    %s\n\n", msg)
            fmt.Println("\033[90m────────────────────────────\033[0m")
        }
        return nil
    },
}

var StashCmd = &cobra.Command{
    Use:   "stash",
    Short: "Save staged files to stash and clear staging area",
    RunE: func(cmd *cobra.Command, args []string) error {
        if !state.Exists() { return fmt.Errorf("not a devflow workspace") }
        s, _ := state.Load()
        if err := state.StashFiles(); err != nil { return err }
        output.Success(fmt.Sprintf("Stashed %d file(s)", len(s.Staged)))
        for _, p := range s.Staged { output.Dim("  stashed: " + p) }
        return nil
    },
}

var UnstashCmd = &cobra.Command{
    Use:   "unstash",
    Short: "Restore stashed files back to the staging area",
    RunE: func(cmd *cobra.Command, args []string) error {
        if !state.Exists() { return fmt.Errorf("not a devflow workspace") }
        s, _ := state.Load()
        if err := state.UnstashFiles(); err != nil { return err }
        output.Success(fmt.Sprintf("Restored %d file(s) from stash", len(s.Stash)))
        for _, p := range s.Stash { output.Info("  restored: " + p) }
        return nil
    },
}

func init() {
	CommitCmd.Flags().StringP("message", "m", "", "Commit message")
	CheckoutCmd.Flags().BoolP("b", "b", false, "Create new branch")
}
