package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"quickpod-cli/internal/app"
)

func newSecurityCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "security",
		Short: "Manage account security and two-factor authentication",
	}

	cmd.AddCommand(newSecurity2FAStatusCmd())
	cmd.AddCommand(newSecurityEnableEmail2FACmd())
	cmd.AddCommand(newSecuritySetupTOTPCmd())
	cmd.AddCommand(newSecurityEnableTOTPCmd())
	cmd.AddCommand(newSecurityDisable2FACmd())
	return cmd
}

func newSecurity2FAStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "2fa-status",
		Short: "Show two-factor authentication status",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireAuth(); err != nil {
				return err
			}
			ctx := context.Background()
			var response map[string]any
			if err := getJSON(ctx, "/update/auth/2fa", true, &response); err != nil {
				return err
			}
			return renderTableOrJSON(response, []string{"KEY", "VALUE"}, [][]string{{"enabled", app.StringValue(response["enabled"])}, {"method", app.StringValue(response["method"])}, {"email_verified", app.StringValue(response["email_verified"])}, {"totp_configured", app.StringValue(response["totp_configured"])}})
		},
	}
	return cmd
}

func newSecurityEnableEmail2FACmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "enable-email-2fa",
		Short: "Enable email-based two-factor authentication",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireAuth(); err != nil {
				return err
			}
			ctx := context.Background()
			var response map[string]any
			if err := postJSON(ctx, "/update/auth/2fa/email/enable", map[string]any{}, true, &response); err != nil {
				return err
			}
			return renderTableOrJSON(response, []string{"KEY", "VALUE"}, [][]string{{"enabled", app.StringValue(response["enabled"])}, {"method", app.StringValue(response["method"])}})
		},
	}
	return cmd
}

func newSecuritySetupTOTPCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "setup-totp",
		Short: "Generate a TOTP secret and otpauth URL",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireAuth(); err != nil {
				return err
			}
			ctx := context.Background()
			var response map[string]any
			if err := postJSON(ctx, "/update/auth/2fa/totp/setup", map[string]any{}, true, &response); err != nil {
				return err
			}
			return renderTableOrJSON(response, []string{"KEY", "VALUE"}, [][]string{{"secret", app.StringValue(response["secret"])}, {"otpauth_url", app.StringValue(response["otpauth_url"])}, {"pending_activation", app.StringValue(response["pending_activation"])}})
		},
	}
	return cmd
}

func newSecurityEnableTOTPCmd() *cobra.Command {
	var code string
	cmd := &cobra.Command{
		Use:   "enable-totp",
		Short: "Enable TOTP two-factor authentication using a current code",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireAuth(); err != nil {
				return err
			}
			if code == "" {
				return fmt.Errorf("--code is required")
			}
			ctx := context.Background()
			var response map[string]any
			if err := postJSON(ctx, "/update/auth/2fa/totp/enable", map[string]any{"code": code}, true, &response); err != nil {
				return err
			}
			return renderTableOrJSON(response, []string{"KEY", "VALUE"}, [][]string{{"enabled", app.StringValue(response["enabled"])}, {"method", app.StringValue(response["method"])}})
		},
	}
	cmd.Flags().StringVar(&code, "code", "", "current 6-digit TOTP code")
	return cmd
}

func newSecurityDisable2FACmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "disable-2fa",
		Short: "Disable two-factor authentication",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireAuth(); err != nil {
				return err
			}
			ctx := context.Background()
			var response map[string]any
			if err := postJSON(ctx, "/update/auth/2fa/disable", map[string]any{}, true, &response); err != nil {
				return err
			}
			return renderTableOrJSON(response, []string{"KEY", "VALUE"}, [][]string{{"enabled", app.StringValue(response["enabled"])}, {"method", app.StringValue(response["method"])}})
		},
	}
	return cmd
}