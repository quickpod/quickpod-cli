package cmd

import (
	"bufio"
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"quickpod-cli/internal/app"
)

type loginResponse struct {
	AuthToken         string `json:"authToken"`
	TwoFactorRequired bool   `json:"two_factor_required"`
	TwoFactorMethod   string `json:"two_factor_method"`
	Message           string `json:"message"`
	Error             string `json:"error"`
}

type oauthExchangeResponse struct {
	AuthToken         string `json:"authToken"`
	Result            string `json:"result"`
	ID                int64  `json:"id"`
	Provider          string `json:"provider"`
	IsNewUser         bool   `json:"is_new_user"`
	SignupRequired    bool   `json:"signup_required"`
	SignupToken       string `json:"signup_token"`
	Email             string `json:"email"`
	Name              string `json:"name"`
	EmailVerified     bool   `json:"email_verified"`
	AvatarURL         string `json:"avatar_url"`
	TwoFactorRequired bool   `json:"two_factor_required"`
	TwoFactorMethod   string `json:"two_factor_method"`
	Message           string `json:"message"`
	Error             string `json:"error"`
}

func newAuthCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Authenticate and manage the local QuickPod session",
	}

	cmd.AddCommand(newAuthLoginCmd())
	cmd.AddCommand(newAuthOAuthLoginCmd("google"))
	cmd.AddCommand(newAuthOAuthLoginCmd("github"))
	cmd.AddCommand(newAuthSignupCmd())
	cmd.AddCommand(newAuthMeCmd())
	cmd.AddCommand(newAuthLogoutCmd())
	cmd.AddCommand(newAuthTokenCmd())
	cmd.AddCommand(newAuthSetTokenCmd())

	return cmd
}

func newAuthOAuthLoginCmd(provider string) *cobra.Command {
	var code string
	var redirectURI string
	var userType string
	var affiliatedWith string
	var twoFactorCode string
	var clientID string
	var printAuthURL bool

	cmd := &cobra.Command{
		Use:   provider,
		Short: fmt.Sprintf("Log in with %s OAuth", strings.Title(provider)),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()

			if printAuthURL {
				authURL, err := buildOAuthAuthorizationURL(provider, clientID, redirectURI)
				if err != nil {
					return err
				}
				if runtimeConfig.Output == "json" {
					return app.PrintJSON(map[string]string{"provider": provider, "authorization_url": authURL})
				}
				fmt.Fprintln(os.Stdout, authURL)
				return nil
			}

			if strings.TrimSpace(code) == "" {
				return fmt.Errorf("--code is required; use --print-auth-url if you need the provider authorization URL")
			}
			if strings.TrimSpace(redirectURI) == "" {
				return fmt.Errorf("--redirect-uri is required")
			}

			response, err := performOAuthLogin(ctx, provider, code, redirectURI, userType, affiliatedWith, twoFactorCode)
			if err != nil {
				return err
			}

			runtimeConfig.Token = response.AuthToken
			if err := saveRuntimeConfig(); err != nil {
				return err
			}
			apiClient.SetToken(response.AuthToken)

			rows := [][]string{
				{"status", firstNonEmpty(response.Result, "OAuth login successful")},
				{"provider", firstNonEmpty(response.Provider, provider)},
				{"id", fmt.Sprintf("%d", response.ID)},
				{"is_new_user", boolString(response.IsNewUser)},
			}
			if strings.TrimSpace(response.TwoFactorMethod) != "" {
				rows = append(rows, []string{"2fa", response.TwoFactorMethod})
			}

			return renderTableOrJSON(map[string]any{
				"status":      firstNonEmpty(response.Result, "OAuth login successful"),
				"provider":    firstNonEmpty(response.Provider, provider),
				"id":          response.ID,
				"is_new_user": response.IsNewUser,
				"2fa":         firstNonEmpty(response.TwoFactorMethod, "none"),
			}, []string{"KEY", "VALUE"}, rows)
		},
	}

	cmd.Flags().StringVar(&code, "code", "", "OAuth authorization code returned by the provider")
	cmd.Flags().StringVar(&redirectURI, "redirect-uri", "", "redirect URI used in the OAuth authorization request")
	cmd.Flags().StringVar(&userType, "user-type", "", "required for first-time OAuth signup: user or host")
	cmd.Flags().StringVar(&affiliatedWith, "affiliated-with", "", "optional affiliation code for first-time OAuth signup")
	cmd.Flags().StringVar(&twoFactorCode, "two-factor-code", "", "two-factor code for email or TOTP login")
	cmd.Flags().StringVar(&clientID, "client-id", "", "OAuth client ID used only for --print-auth-url")
	cmd.Flags().BoolVar(&printAuthURL, "print-auth-url", false, "print the provider authorization URL instead of exchanging a code")
	return cmd
}

func newAuthLoginCmd() *cobra.Command {
	var email string
	var password string
	var twoFactorCode string

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Log in with email and password, including two-factor challenges when required",
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

			response, err := performLogin(ctx, email, password, twoFactorCode)
			if err != nil {
				return err
			}

			runtimeConfig.Token = response.AuthToken
			if err := saveRuntimeConfig(); err != nil {
				return err
			}
			apiClient.SetToken(response.AuthToken)

			rows := [][]string{{"status", "logged in"}, {"base_url", runtimeConfig.BaseURL}}
			if strings.TrimSpace(response.TwoFactorMethod) != "" {
				rows = append(rows, []string{"2fa", response.TwoFactorMethod})
			}

			return renderTableOrJSON(map[string]string{
				"status":   "logged in",
				"base_url": runtimeConfig.BaseURL,
				"2fa":      firstNonEmpty(response.TwoFactorMethod, "none"),
			}, []string{"KEY", "VALUE"}, rows)
		},
	}

	cmd.Flags().StringVar(&email, "email", "", "QuickPod account email")
	cmd.Flags().StringVar(&password, "password", "", "QuickPod password; if omitted you will be prompted")
	cmd.Flags().StringVar(&twoFactorCode, "two-factor-code", "", "two-factor code for email or TOTP login; if omitted, interactive login will prompt when required")
	return cmd
}

func performLogin(ctx context.Context, email string, password string, twoFactorCode string) (*loginResponse, error) {
	response, err := sendLoginRequest(ctx, email, password, twoFactorCode)
	if err != nil {
		return nil, err
	}

	if strings.TrimSpace(response.AuthToken) != "" {
		return response, nil
	}

	if !response.TwoFactorRequired {
		if response.Error != "" {
			return nil, fmt.Errorf(response.Error)
		}
		if response.Message != "" {
			return nil, fmt.Errorf(response.Message)
		}
		return nil, fmt.Errorf("login succeeded but no auth token was returned")
	}

	method := firstNonEmpty(response.TwoFactorMethod, "unknown")
	message := firstNonEmpty(response.Message, "Two-factor verification is required")
	if strings.TrimSpace(twoFactorCode) != "" {
		return nil, fmt.Errorf("two-factor login challenge for %s was not completed: %s", method, message)
	}

	fmt.Fprintf(os.Stdout, "%s (%s)\n", message, method)
	code, err := promptSecret("Two-factor code: ")
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(code) == "" {
		return nil, fmt.Errorf("two-factor code is required to complete login")
	}

	response, err = sendLoginRequest(ctx, email, password, code)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(response.AuthToken) == "" {
		if response.Error != "" {
			return nil, fmt.Errorf(response.Error)
		}
		if response.Message != "" {
			return nil, fmt.Errorf(response.Message)
		}
		return nil, fmt.Errorf("two-factor verification completed but no auth token was returned")
	}
	response.TwoFactorMethod = method
	return response, nil
}

func sendLoginRequest(ctx context.Context, email string, password string, twoFactorCode string) (*loginResponse, error) {
	requestBody := map[string]any{
		"email":    email,
		"password": password,
	}
	if strings.TrimSpace(twoFactorCode) != "" {
		requestBody["two_factor_code"] = strings.TrimSpace(twoFactorCode)
	}

	var response loginResponse
	if err := postJSON(ctx, "/update/auth/login", requestBody, false, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

func performOAuthLogin(ctx context.Context, provider string, code string, redirectURI string, userType string, affiliatedWith string, twoFactorCode string) (*oauthExchangeResponse, error) {
	response, err := sendOAuthExchangeRequest(ctx, provider, map[string]any{
		"code":            strings.TrimSpace(code),
		"redirect_uri":    strings.TrimSpace(redirectURI),
		"user_type":       strings.TrimSpace(userType),
		"affiliatedwith":  strings.TrimSpace(affiliatedWith),
		"two_factor_code": strings.TrimSpace(twoFactorCode),
	})
	if err != nil {
		return nil, err
	}

	if response.SignupRequired {
		chosenType := strings.TrimSpace(userType)
		if chosenType == "" {
			fmt.Fprintf(os.Stdout, "OAuth signup required for %s (%s)\n", firstNonEmpty(response.Email, response.Name), provider)
			promptedType, promptErr := promptLine("Account type [user/host]: ")
			if promptErr != nil {
				return nil, promptErr
			}
			chosenType = promptedType
		}
		chosenType = strings.ToLower(strings.TrimSpace(chosenType))
		if chosenType != "user" && chosenType != "host" {
			return nil, fmt.Errorf("invalid user type %q; use user or host", chosenType)
		}

		response, err = sendOAuthExchangeRequest(ctx, provider, map[string]any{
			"signup_token":    response.SignupToken,
			"user_type":       chosenType,
			"affiliatedwith":  strings.TrimSpace(affiliatedWith),
			"two_factor_code": strings.TrimSpace(twoFactorCode),
		})
		if err != nil {
			return nil, err
		}
		userType = chosenType
	}

	if strings.TrimSpace(response.AuthToken) != "" {
		return response, nil
	}

	if response.TwoFactorRequired {
		method := firstNonEmpty(response.TwoFactorMethod, "unknown")
		message := firstNonEmpty(response.Message, "Two-factor verification is required")
		if strings.TrimSpace(twoFactorCode) != "" {
			return nil, fmt.Errorf("two-factor login challenge for %s was not completed: %s", method, message)
		}
		fmt.Fprintf(os.Stdout, "%s (%s)\n", message, method)
		codePrompt, promptErr := promptSecret("Two-factor code: ")
		if promptErr != nil {
			return nil, promptErr
		}
		if strings.TrimSpace(codePrompt) == "" {
			return nil, fmt.Errorf("two-factor code is required to complete oauth login")
		}

		payload := map[string]any{
			"code":            strings.TrimSpace(code),
			"redirect_uri":    strings.TrimSpace(redirectURI),
			"user_type":       strings.TrimSpace(userType),
			"affiliatedwith":  strings.TrimSpace(affiliatedWith),
			"two_factor_code": strings.TrimSpace(codePrompt),
		}
		if strings.TrimSpace(response.SignupToken) != "" {
			payload = map[string]any{
				"signup_token":    strings.TrimSpace(response.SignupToken),
				"user_type":       strings.TrimSpace(firstNonEmpty(userType, "user")),
				"affiliatedwith":  strings.TrimSpace(affiliatedWith),
				"two_factor_code": strings.TrimSpace(codePrompt),
			}
		}

		response, err = sendOAuthExchangeRequest(ctx, provider, payload)
		if err != nil {
			return nil, err
		}
		response.TwoFactorMethod = method
	}

	if strings.TrimSpace(response.AuthToken) == "" {
		if response.Error != "" {
			return nil, fmt.Errorf(response.Error)
		}
		if response.Message != "" {
			return nil, fmt.Errorf(response.Message)
		}
		return nil, fmt.Errorf("oauth login did not return an auth token")
	}

	return response, nil
}

func sendOAuthExchangeRequest(ctx context.Context, provider string, payload map[string]any) (*oauthExchangeResponse, error) {
	endpoint := "/update/auth/oauth/" + provider + "/exchange"
	var response oauthExchangeResponse
	if err := postJSON(ctx, endpoint, payload, false, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

func buildOAuthAuthorizationURL(provider string, clientID string, redirectURI string) (string, error) {
	resolvedClientID := strings.TrimSpace(clientID)
	if resolvedClientID == "" {
		switch provider {
		case "google":
			resolvedClientID = strings.TrimSpace(os.Getenv("QUICKPOD_GOOGLE_CLIENT_ID"))
		case "github":
			resolvedClientID = strings.TrimSpace(os.Getenv("QUICKPOD_GITHUB_CLIENT_ID"))
		}
	}
	if resolvedClientID == "" {
		return "", fmt.Errorf("client ID is required for %s auth URL generation; pass --client-id or set the matching QUICKPOD_*_CLIENT_ID env var", provider)
	}
	if strings.TrimSpace(redirectURI) == "" {
		return "", fmt.Errorf("--redirect-uri is required")
	}

	switch provider {
	case "google":
		query := url.Values{}
		query.Set("client_id", resolvedClientID)
		query.Set("redirect_uri", redirectURI)
		query.Set("response_type", "code")
		query.Set("scope", "openid email profile")
		query.Set("access_type", "offline")
		query.Set("prompt", "consent")
		return "https://accounts.google.com/o/oauth2/v2/auth?" + query.Encode(), nil
	case "github":
		query := url.Values{}
		query.Set("client_id", resolvedClientID)
		query.Set("redirect_uri", redirectURI)
		query.Set("scope", "read:user user:email")
		return "https://github.com/login/oauth/authorize?" + query.Encode(), nil
	default:
		return "", fmt.Errorf("unsupported oauth provider %q", provider)
	}
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
				"name":           name,
				"email":          email,
				"password":       password,
				"user_type":      userType,
				"affiliatedwith": affiliatedWith,
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

	if prompt != "" {
		fmt.Fprint(os.Stdout, prompt)
	}
	reader := bufio.NewReader(os.Stdin)
	value, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(value), nil
}

func promptLine(prompt string) (string, error) {
	if prompt != "" {
		fmt.Fprint(os.Stdout, prompt)
	}
	reader := bufio.NewReader(os.Stdin)
	value, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(value), nil
}