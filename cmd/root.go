package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"quickpod-cli/internal/app"
)

var (
	configPath      string
	baseURLOverride string
	tokenOverride   string
	apiKeyOverride  string
	outputOverride  string

	runtimeConfig app.Config
	apiClient     *app.Client
)

var rootCmd = &cobra.Command{
	Use:   "quickpod",
	Short: "QuickPod GPU and CPU platform CLI",
	Long: `QuickPod is a Go CLI for the public and user-facing QuickPod GPU and CPU platform APIs.

It covers machine discovery, pods, templates, user account workflows, host machine listing,
user storage volumes, host store metadata, and user security operations.`,
	Example: `  quickpod auth login --email you@example.com
	quickpod search gpu --type A100 --max-hourly 2.5 --limit 10
	quickpod pods list --kind gpu
	quickpod templates list --scope my --kind gpu
	quickpod machines list --kind gpu
	quickpod storage volumes list
	quickpod account affiliations`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return initRuntime()
	},
	SilenceUsage:  true,
	SilenceErrors: true,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	defaultConfigPath, err := app.DefaultConfigPath()
	if err != nil {
		defaultConfigPath = "./quickpod-cli-config.json"
	}

	rootCmd.PersistentFlags().StringVar(&configPath, "config", defaultConfigPath, "config file path")
	rootCmd.PersistentFlags().StringVar(&baseURLOverride, "base-url", "", "override the QuickPod API base URL")
	rootCmd.PersistentFlags().StringVar(&tokenOverride, "token", "", "override the stored auth token or API key for this invocation")
	rootCmd.PersistentFlags().StringVar(&apiKeyOverride, "api-key", "", "override the stored secure API key for this invocation")
	rootCmd.PersistentFlags().StringVarP(&outputOverride, "output", "o", "", "output format: table or json")

	rootCmd.AddCommand(newAuthCmd())
	rootCmd.AddCommand(newSearchCmd())
	rootCmd.AddCommand(newCatalogCmd())
	rootCmd.AddCommand(newPodsCmd())
	rootCmd.AddCommand(newClustersCmd())
	rootCmd.AddCommand(newServerlessCmd())
	rootCmd.AddCommand(newTemplatesCmd())
	rootCmd.AddCommand(newMachinesCmd())
	rootCmd.AddCommand(newStorageCmd())
	rootCmd.AddCommand(newAccountCmd())
	rootCmd.AddCommand(newSecurityCmd())
	rootCmd.AddCommand(newStoreCmd())
	rootCmd.AddCommand(newCompletionCmd())
	rootCmd.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Print the CLI version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintln(os.Stdout, "quickpod-cli dev")
		},
	})
}

func initRuntime() error {
	config, err := app.LoadConfig(configPath)
	if err != nil {
		return err
	}

	if envBaseURL := strings.TrimSpace(os.Getenv("QUICKPOD_BASE_URL")); envBaseURL != "" {
		config.BaseURL = envBaseURL
	}
	if envToken := strings.TrimSpace(os.Getenv("QUICKPOD_TOKEN")); envToken != "" {
		config.Token = envToken
	}
	if envAPIKey := strings.TrimSpace(os.Getenv("QUICKPOD_API_KEY")); envAPIKey != "" {
		config.Token = envAPIKey
	}
	if envOutput := strings.TrimSpace(os.Getenv("QUICKPOD_OUTPUT")); envOutput != "" {
		config.Output = envOutput
	}

	if strings.TrimSpace(baseURLOverride) != "" {
		config.BaseURL = baseURLOverride
	}
	if strings.TrimSpace(tokenOverride) != "" {
		config.Token = tokenOverride
	}
	if strings.TrimSpace(apiKeyOverride) != "" {
		config.Token = apiKeyOverride
	}
	if strings.TrimSpace(outputOverride) != "" {
		config.Output = outputOverride
	}

	normalizedBaseURL, err := app.NormalizeBaseURL(config.BaseURL)
	if err != nil {
		return err
	}
	config.BaseURL = normalizedBaseURL
	config.Output = strings.ToLower(strings.TrimSpace(config.Output))
	if config.Output == "" {
		config.Output = "table"
	}
	if config.Output != "table" && config.Output != "json" {
		return fmt.Errorf("unsupported output format %q", config.Output)
	}

	runtimeConfig = config
	apiClient = app.NewClient(runtimeConfig.BaseURL, runtimeConfig.Token)
	return nil
}

func saveRuntimeConfig() error {
	return app.SaveConfig(configPath, runtimeConfig)
}

func newCompletionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate shell completion",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "bash":
				return rootCmd.GenBashCompletion(os.Stdout)
			case "zsh":
				return rootCmd.GenZshCompletion(os.Stdout)
			case "fish":
				return rootCmd.GenFishCompletion(os.Stdout, true)
			case "powershell":
				return rootCmd.GenPowerShellCompletionWithDesc(os.Stdout)
			default:
				return fmt.Errorf("unsupported shell %q", args[0])
			}
		},
	}
	return cmd
}
