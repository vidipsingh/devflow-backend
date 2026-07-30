package commands

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"syscall"

	"devflow-backend/internal/cli/client"
	"devflow-backend/internal/cli/config"
	"devflow-backend/internal/cli/output"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var AuthCmd = &cobra.Command{
	Use: "auth",
	Short: "Authenticate with Devflow",
}

var loginCmd = &cobra.Command{
	Use: "login",
	Short: "Log in to your DevFlow account",
	RunE: func(cmd *cobra.Command, args []string) error {
		reader := bufio.NewReader(os.Stdin)

		fmt.Print("Email: ")
		email, _ := reader.ReadString('\n')
		email = strings.TrimSpace(email)

		fmt.Print("Password: ")
		raw, err := term.ReadPassword(int(syscall.Stdin))
		fmt.Println()
		if err != nil {
			return err
		}
		password := string(raw)

		c, err := client.New(false)
        if err != nil {
            return err
        }
		data, err := c.Post("/auth/login", map[string]string{
			"email": email,
			"password": password,
		})
		if err != nil {
			return fmt.Errorf("login failed: %w", err)
		}
		m, _ := data.(map[string]any)
		token, _ := m["token"].(string)
		user, _ := m["user"].(map[string]any)
		username, _ := user["username"].(string)

		cfg, _ := config.Load()
		cfg.Token = token
		if err := config.Save(cfg); err != nil {
			return err
		}
		output.Success(fmt.Sprintf("Logged in as %s", username))
		return nil
	},
}

var logoutCmd = &cobra.Command{
	Use: "logout",
	Short: "Clear saved credentials",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := config.ClearToken(); err != nil {
			return err
		}
		output.Success("Logged out")
		return nil
	},
}

var whoamiCmd = &cobra.Command{
	Use: "whoami",
	Short: "Show current logged-in user",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := client.New(true)
		if err != nil {
			return err
		}
		data, err := c.Get("/me")
		if err != nil {
			return err
		}
		m, _ := data.(map[string]any)
		output.Bold(fmt.Sprintf("Username : %v", m["username"]))
		output.Bold(fmt.Sprintf("Email : %v", m["email"]))
		output.Bold(fmt.Sprintf("Plan : %v", m["plan"]))
		return nil
	},
}

func mustHost() string {
	cfg, _ := config.Load()
	if cfg.Host == "" {
		return "http://localhost:8080"
	}
	return cfg.Host
}

func init() {
	AuthCmd.AddCommand(loginCmd, logoutCmd, whoamiCmd)
}