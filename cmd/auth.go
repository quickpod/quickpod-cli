package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"quickpod-cli/internal/app"
)

func newAuthCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Authenticate and manage the local QuickPod session",
	}

	cmd.AddCommand(newAuthLoginCmd())
	cmd.AddCommand(newAuthSignupCmd())
	cmd.AddCommand(newAuthMeCmd())
	cmd.AddCommand(newAuthLogoutCmd())
	cmd.AddCommand(newAuthTokenCmd())
	cmd.AddCommand(newAuthSetTokenCmd())

	return cmd
}

func newAuthLoginCmd() *cobra.Command {
	var email string
	var password string
	var twoFactorCode string

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Log in with email and password and store the returned auth token",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			if strings.TrimSpace(email) == "" {
				return fmt.Errorf("--email is required")
			}
			if strings.TrimSpace(password) == "" {
				secret, err := promptSecret("Password: ")
				if err != nil {
					return err
				}
				password = secret
			}

			var response struct {
				AuthToken string `json:"authToken"`
			}
			requestBody := map[string]any{
				"email":    email,
				"password": password,
			}
			if strings.TrimSpace(twoFactorCode) != "" {
				requestBody["two_factor_code"] = twoFactorCode
			}

			if err := postJSON(ctx, "/update/auth/login", requestBody, false, &response); err != nil {
				return err
			}
			if strings.TrimSpace(response.AuthToken) == "" {
				return fmt.Errorf("login succeeded but no auth token was returned")
			}

			runtimeConfig.Token = response.AuthToken
			if err := saveRuntimeConfig(); err != nil {
				return err
			}
			apiClient.SetToken(response.AuthToken)
			return renderTableOrJSON(map[string]string{
				"status":   "logged in",
				"base_url": runtimeConfig.BaseURL,
			}, []string{"KEY", "VALUE"}, [][]string{{"status", "logged in"}, {"base_url", runtimeConfig.BaseURL}})
		},
	}

	cmd.Flags().StringVar(&email, "email", "", "QuickPod account email")
	cmd.Flags().StringVar(&password, "password", "", "QuickPod password; if omitted you will be prompted")
	cmd.Flags().StringVar(&twoFactorCode, "two-factor-code", "", "two-factor code for email or TOTP login")
	return cmd
}

func newAuthSignupCmd() *cobra.Command {
	var name string
	var email string
	var password string
	var userType string
	var affiliatedWith string

	cmd := &cobra.Command{
		Use:   "signup",
		Short: "Create a QuickPod account and store the returned auth token when available",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			if strings.TrimSpace(name) == "" || strings.TrimSpace(email) == "" {
				return fmt.Errorf("--name and --email are required")
			}
			if strings.TrimSpace(password) == "" {
				secret, err := promptSecret("Password: ")
				if err != nil {
					return err
				}
				password = secret
			}

			var response map[string]any
			requestBody := map[string]any{
				"name":            name,
				"email":           email,
				"password":        password,
				"user_type":       userType,
				"affiliatedwith":  affiliatedWith,
			}
			if err := postJSON(ctx, "/update/auth/signup", requestBody, false, &response); err != nil {
				return err
			}

			token := app.StringValue(response["authToken"])
			if token != "" {
				runtimeConfig.Token = token
				if err := saveRuntimeConfig(); err != nil {
					return err
				}
				apiClient.SetToken(token)
			}

			return renderTableOrJSON(response, []string{"KEY", "VALUE"}, [][]string{
				{"result", firstNonEmpty(app.StringValue(response["result"]), "signup successful")},
				{"id", app.StringValue(response["id"])},
				{"token_stored", boolString(token != "")},
			})
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "display name")
	cmd.Flags().StringVar(&email, "email", "", "QuickPod account email")
	cmd.Flags().StringVar(&password, "password", "", "password; if omitted you will be prompted")
	cmd.Flags().StringVar(&userType, "user-type", "user", "account type: user or host")
	cmd.Flags().StringVar(&affiliatedWith, "affiliated-with", "", "optional affiliation code or source")
	return cmd
}

func newAuthMeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "me",
		Short: "Show the currently authenticated user profile",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireAuth(); err != nil {
				return err
			}
			ctx := context.Background()
			var response map[string]any
			if err := getJSON(ctx, "/update/auth/me", true, &response); err != nil {
				return err
			}
			rows := [][]string{
				{"id", app.StringValue(response["id"])},
				{"name", app.StringValue(response["name"])},
				{"email", app.StringValue(response["email"])},
				{"user_type", app.StringValue(response["user_type"])},
				{"credit", app.StringValue(response["credit"])},
				{"api_key", app.StringValue(response["api_key"])},
			}
			return renderTableOrJSON(response, []string{"KEY", "VALUE"}, rows)
		},
	}
	return cmd
}

func newAuthLogoutCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logout",
		Short: "Remove the locally stored auth token",
		RunE: func(cmd *cobra.Command, args []string) error {
			runtimeConfig.Token = ""
			if err := saveRuntimeConfig(); err != nil {
				return err
			}
			apiClient.SetToken("")
			return renderTableOrJSON(map[string]string{"status": "logged out"}, []string{"KEY", "VALUE"}, [][]string{{"status", "logged out"}})
		},
	}
	return cmd
}

func newAuthTokenCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "token",
		Short: "Print the active auth token",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireAuth(); err != nil {
				return err
			}
			if runtimeConfig.Output == "json" {
				return app.PrintJSON(map[string]string{"token": runtimeConfig.Token})
			}
			fmt.Fprintln(os.Stdout, runtimeConfig.Token)
			return nil
		},
	}
	return cmd
}

func newAuthSetTokenCmd() *cobra.Command {
	var token string
	cmd := &cobra.Command{
		Use:   "set-token",
		Short: "Store an existing auth token without logging in",
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(token) == "" {
				reader := bufio.NewReader(os.Stdin)
				fmt.Fprint(os.Stdout, "Token: ")
				value, err := reader.ReadString('\n')
				if err != nil {
					return err
				}
				token = strings.TrimSpace(value)
			}
			if strings.TrimSpace(token) == "" {
				return fmt.Errorf("token is required")
			}
			runtimeConfig.Token = strings.TrimSpace(token)
			if err := saveRuntimeConfig(); err != nil {
				return err
			}
			apiClient.SetToken(runtimeConfig.Token)
			return renderTableOrJSON(map[string]string{"status": "token stored"}, []string{"KEY", "VALUE"}, [][]string{{"status", "token stored"}})
		},
	}
	cmd.Flags().StringVar(&token, "value", "", "auth token to store")
	return cmd
}

func promptSecret(prompt string) (string, error) {
	if term.IsTerminal(int(os.Stdin.Fd())) {
		fmt.Fprint(os.Stdout, prompt)
		secret, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Fprintln(os.Stdout)
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(secret)), nil
	}

	reader := bufio.NewReader(os.Stdin)
	value, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(value), nil
}