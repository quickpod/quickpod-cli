package cmd

import (
	"context"
	"fmt"
	"net/url"

	"github.com/spf13/cobra"

	"quickpod-cli/internal/app"
)

func newAccountCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "account",
		Short: "Inspect account metrics and run user account operations",
	}

	cmd.AddCommand(newPublicAccountGetCmd("indicators", "/key_indicators", []string{"METRIC", "VALUE"}))
	cmd.AddCommand(newPublicAccountListCmd("monthly-payments", "/monthly_payments", []string{"MONTH", "AMOUNT"}, "yearmonth", "amount"))
	cmd.AddCommand(newPublicAccountListCmd("monthly-spending", "/monthly_spending", []string{"MONTH", "AMOUNT"}, "yearmonth", "amount"))
	cmd.AddCommand(newPublicAccountListCmd("monthly-payouts", "/monthly_payouts", []string{"MONTH", "AMOUNT"}, "yearmonth", "amount"))
	cmd.AddCommand(newPublicAccountListCmd("billing-rate-history", "/billing_rate_history", []string{"TIMESTAMP", "BILLING_RATE"}, "timestamp", "billing_rate"))
	cmd.AddCommand(newAuthedAccountListCmd("transactions", "/update/transactions", []string{"ID", "GATEWAY", "STATUS", "AMOUNT", "CREATED"}, "id", "gateway", "payment_status", "amount_total", "created_at"))
	cmd.AddCommand(newAuthedAccountListCmd("affiliations", "/update/affiliations", []string{"ID", "EMAIL", "TOTAL_BILLED", "EARNINGS"}, "id", "email", "total_billed", "earnings"))
	cmd.AddCommand(newAuthedAccountListCmd("audit-log", "/audit_log", []string{"ID", "METHOD", "PATH", "STATUS", "TIMESTAMP"}, "id", "http_method", "full_path", "response_code", "timestamp"))
	cmd.AddCommand(newAuthedAccountListCmd("host-earnings", "/host_earnings_history", []string{"CREATED", "EARN/HOUR", "STORAGE/HOUR"}, "created_at", "earnings_per_hour", "storage_per_hour"))
	cmd.AddCommand(newAccountEmailCheckCmd())
	cmd.AddCommand(newAccountContactCmd())
	cmd.AddCommand(newAccountResetAPIKeyCmd())
	cmd.AddCommand(newAccountReverifyEmailCmd())

	return cmd
}

func newPublicAccountGetCmd(use, endpoint string, headers []string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   use,
		Short: "Fetch public account metrics",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			var response map[string]any
			if err := getJSON(ctx, endpoint, false, &response); err != nil {
				return err
			}
			rows := make([][]string, 0, len(response))
			for key, value := range response {
				rows = append(rows, []string{key, app.StringValue(value)})
			}
			return renderTableOrJSON(response, headers, rows)
		},
	}
	return cmd
}

func newPublicAccountListCmd(use, endpoint string, headers []string, keys ...string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   use,
		Short: "Fetch public time-series data",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			var items []map[string]any
			if err := getJSON(ctx, endpoint, false, &items); err != nil {
				return err
			}
			return renderTableOrJSON(items, headers, genericRows(items, keys...))
		},
	}
	return cmd
}

func newAuthedAccountListCmd(use, endpoint string, headers []string, keys ...string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   use,
		Short: "Fetch authenticated account data",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireAuth(); err != nil {
				return err
			}
			ctx := context.Background()
			var items []map[string]any
			if err := getJSON(ctx, endpoint, true, &items); err != nil {
				return err
			}
			return renderTableOrJSON(items, headers, genericRows(items, keys...))
		},
	}
	return cmd
}

func newAccountEmailCheckCmd() *cobra.Command {
	var email string
	cmd := &cobra.Command{
		Use:   "email-check",
		Short: "Check whether an email exists and still needs a password hash",
		RunE: func(cmd *cobra.Command, args []string) error {
			if email == "" {
				return fmt.Errorf("--email is required")
			}
			ctx := context.Background()
			query := url.Values{}
			query.Set("email", email)
			var response map[string]any
			if err := getJSONQuery(ctx, "/email_check", query, false, &response); err != nil {
				return err
			}
			return renderTableOrJSON(response, []string{"KEY", "VALUE"}, [][]string{{"email", email}, {"exists", app.StringValue(response["exists"])}})
		},
	}
	cmd.Flags().StringVar(&email, "email", "", "email address to check")
	return cmd
}

func newAccountContactCmd() *cobra.Command {
	var name string
	var email string
	var company string
	var message string
	cmd := &cobra.Command{
		Use:   "contact",
		Short: "Send a contact request",
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" || email == "" || message == "" {
				return fmt.Errorf("--name, --email, and --message are required")
			}
			ctx := context.Background()
			var response map[string]any
			if err := postJSON(ctx, "/update/contact", map[string]any{"name": name, "email": email, "company": company, "message": message}, false, &response); err != nil {
				return err
			}
			return renderTableOrJSON(response, []string{"KEY", "VALUE"}, [][]string{{"result", firstNonEmpty(app.StringValue(response["result"]), app.StringValue(response["message"]))}})
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "contact name")
	cmd.Flags().StringVar(&email, "email", "", "contact email")
	cmd.Flags().StringVar(&company, "company", "", "company name")
	cmd.Flags().StringVar(&message, "message", "", "message body")
	return cmd
}

func newAccountResetAPIKeyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reset-api-key",
		Short: "Reset and fetch your user API key",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireAuth(); err != nil {
				return err
			}
			ctx := context.Background()
			var response map[string]any
			if err := getJSON(ctx, "/update/resetapikey", true, &response); err != nil {
				return err
			}
			return renderTableOrJSON(response, []string{"KEY", "VALUE"}, [][]string{{"api_key", firstNonEmpty(app.StringValue(response["api_key"]), app.StringValue(response["result"]))}})
		},
	}
	return cmd
}

func newAccountReverifyEmailCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reverify-email",
		Short: "Trigger another verification email",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireAuth(); err != nil {
				return err
			}
			ctx := context.Background()
			var response map[string]any
			if err := getJSON(ctx, "/update/reverify_email", true, &response); err != nil {
				return err
			}
			return renderTableOrJSON(response, []string{"KEY", "VALUE"}, [][]string{{"result", firstNonEmpty(app.StringValue(response["result"]), app.StringValue(response["message"]))}})
		},
	}
	return cmd
}