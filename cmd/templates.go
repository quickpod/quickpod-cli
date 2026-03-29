package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"quickpod-cli/internal/app"
)

func newTemplatesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "templates",
		Short: "List and manage QuickPod templates",
	}

	cmd.AddCommand(newTemplatesListCmd())
	cmd.AddCommand(newTemplatesSaveCmd())
	cmd.AddCommand(newTemplatesDeleteCmd())
	cmd.AddCommand(newTemplatesCommunityCmd())

	return cmd
}

func newTemplatesListCmd() *cobra.Command {
	var kind string
	var scope string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List templates by scope and kind",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			endpoint := "/public_templates"
			auth := false

			switch scope {
			case "my":
				auth = true
				if kind == "cpu" {
					endpoint = "/templates_cpu"
				} else {
					endpoint = "/templates"
				}
			case "public":
				if kind == "cpu" {
					endpoint = "/templates_cpu_public"
				} else {
					endpoint = "/public_templates"
				}
			case "community":
				if kind == "cpu" {
					endpoint = "/templates_cpu_community"
				} else {
					endpoint = "/community_templates"
				}
			default:
				return fmt.Errorf("unsupported scope %q", scope)
			}

			if auth {
				if err := requireAuth(); err != nil {
					return err
				}
			}

			var items []map[string]any
			if err := getJSON(ctx, endpoint, auth, &items); err != nil {
				return err
			}
			rows := make([][]string, 0, len(items))
			for _, item := range items {
				rows = append(rows, []string{
					app.StringValue(item["id"]),
					app.Truncate(app.StringValue(item["template_name"]), 26),
					app.StringValue(item["template_type"]),
					app.Truncate(app.StringValue(item["image_path"]), 28),
					boolString(app.BoolValue(item["is_public"])),
					app.Truncate(app.StringValue(item["template_uuid"]), 16),
				})
			}
			return renderTableOrJSON(items, []string{"ID", "NAME", "TYPE", "IMAGE", "PUBLIC", "UUID"}, rows)
		},
	}

	cmd.Flags().StringVar(&kind, "kind", "gpu", "template kind: gpu or cpu")
	cmd.Flags().StringVar(&scope, "scope", "public", "template scope: public, community, or my")
	return cmd
}

func newTemplatesSaveCmd() *cobra.Command {
	var filePath string
	var kind string
	var templateUUID string
	var name string
	var description string
	var imagePath string
	var versionTag string
	var dockerOptions string
	var dockerRunOptions string
	var launchMode string
	var launchPort int
	var onStartScript string
	var extraFilters string
	var dockerRepoServer string
	var dockerRepoUsername string
	var dockerRepoPassword string
	var diskSpace int
	var readme string
	var isPublic bool
	var cliCommand string
	var templateImageURL string
	var envTag string

	cmd := &cobra.Command{
		Use:   "save",
		Short: "Create or update a template from flags or a JSON file",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireAuth(); err != nil {
				return err
			}
			ctx := context.Background()
			payload := map[string]any{}
			if filePath != "" {
				filePayload, err := app.ReadJSONFile(filePath)
				if err != nil {
					return err
				}
				payload = app.MergeMap(payload, filePayload)
			}

			setIfChanged := func(flagName string, key string, value any) {
				if cmd.Flags().Changed(flagName) {
					payload[key] = value
				}
			}

			setIfChanged("kind", "template_type", kind)
			setIfChanged("template-uuid", "template_uuid", templateUUID)
			setIfChanged("name", "template_name", name)
			setIfChanged("description", "template_description", description)
			setIfChanged("image-path", "image_path", imagePath)
			setIfChanged("version", "version_tag", versionTag)
			setIfChanged("docker-options", "docker_options", dockerOptions)
			setIfChanged("docker-run-options", "docker_run_options", dockerRunOptions)
			setIfChanged("launch-mode", "launch_mode", launchMode)
			setIfChanged("launch-port", "launch_port", launchPort)
			setIfChanged("on-start-script", "on_start_script", onStartScript)
			setIfChanged("extra-filters", "extra_filters", extraFilters)
			setIfChanged("docker-repo-server", "docker_repo_server", dockerRepoServer)
			setIfChanged("docker-repo-username", "docker_repo_username", dockerRepoUsername)
			setIfChanged("docker-repo-password", "docker_repo_password", dockerRepoPassword)
			setIfChanged("disk-space", "disk_space", diskSpace)
			setIfChanged("readme", "readme", readme)
			setIfChanged("public", "is_public", isPublic)
			setIfChanged("cli-command", "cli_command", cliCommand)
			setIfChanged("template-image-url", "template_image_url", templateImageURL)
			setIfChanged("env-tag", "env_tag", envTag)

			if len(payload) == 0 {
				return fmt.Errorf("no template fields were provided")
			}

			var response map[string]any
			if err := postJSON(ctx, "/update/templates", payload, true, &response); err != nil {
				return err
			}
			rows := [][]string{
				{"id", app.StringValue(response["id"])},
				{"template_name", app.StringValue(response["template_name"])},
				{"template_uuid", app.StringValue(response["template_uuid"])},
				{"template_type", app.StringValue(response["template_type"])},
				{"is_public", boolString(app.BoolValue(response["is_public"]))},
			}
			return renderTableOrJSON(response, []string{"KEY", "VALUE"}, rows)
		},
	}

	cmd.Flags().StringVar(&filePath, "file", "", "path to a JSON template payload")
	cmd.Flags().StringVar(&kind, "kind", "gpu", "template type: gpu or cpu")
	cmd.Flags().StringVar(&templateUUID, "template-uuid", "", "existing template UUID to update")
	cmd.Flags().StringVar(&name, "name", "", "template name")
	cmd.Flags().StringVar(&description, "description", "", "template description")
	cmd.Flags().StringVar(&imagePath, "image-path", "", "container image path")
	cmd.Flags().StringVar(&versionTag, "version", "", "template version tag")
	cmd.Flags().StringVar(&dockerOptions, "docker-options", "", "docker options")
	cmd.Flags().StringVar(&dockerRunOptions, "docker-run-options", "", "docker run options")
	cmd.Flags().StringVar(&launchMode, "launch-mode", "", "launch mode")
	cmd.Flags().IntVar(&launchPort, "launch-port", 0, "launch port")
	cmd.Flags().StringVar(&onStartScript, "on-start-script", "", "startup script")
	cmd.Flags().StringVar(&extraFilters, "extra-filters", "", "extra filters")
	cmd.Flags().StringVar(&dockerRepoServer, "docker-repo-server", "", "private registry server")
	cmd.Flags().StringVar(&dockerRepoUsername, "docker-repo-username", "", "private registry username")
	cmd.Flags().StringVar(&dockerRepoPassword, "docker-repo-password", "", "private registry password")
	cmd.Flags().IntVar(&diskSpace, "disk-space", 0, "default disk size in GB")
	cmd.Flags().StringVar(&readme, "readme", "", "template readme content")
	cmd.Flags().BoolVar(&isPublic, "public", false, "mark the template public")
	cmd.Flags().StringVar(&cliCommand, "cli-command", "", "CLI launch command")
	cmd.Flags().StringVar(&templateImageURL, "template-image-url", "", "template preview image URL")
	cmd.Flags().StringVar(&envTag, "env-tag", "", "environment tag")
	return cmd
}

func newTemplatesDeleteCmd() *cobra.Command {
	var templateUUID string
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a template by UUID",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireAuth(); err != nil {
				return err
			}
			if templateUUID == "" {
				return fmt.Errorf("--template-uuid is required")
			}
			ctx := context.Background()
			var response map[string]any
			if err := getJSON(ctx, "/update/deleteTemplate/"+templateUUID, true, &response); err != nil {
				return err
			}
			return renderTableOrJSON(response, []string{"KEY", "VALUE"}, [][]string{{"result", firstNonEmpty(app.StringValue(response["result"]), app.StringValue(response["message"]))}})
		},
	}
	cmd.Flags().StringVar(&templateUUID, "template-uuid", "", "template UUID")
	return cmd
}

func newTemplatesCommunityCmd() *cobra.Command {
	var templateID int
	var enabled bool
	cmd := &cobra.Command{
		Use:   "community",
		Short: "Set or unset the community flag for a template you own",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireAuth(); err != nil {
				return err
			}
			if templateID <= 0 {
				return fmt.Errorf("--template-id is required")
			}
			ctx := context.Background()
			var response map[string]any
			if err := postJSON(ctx, "/update/templates/community", map[string]any{"template_id": templateID, "is_community": enabled}, true, &response); err != nil {
				return err
			}
			return renderTableOrJSON(response, []string{"KEY", "VALUE"}, [][]string{{"template_id", app.StringValue(response["template_id"])}, {"is_community", app.StringValue(response["is_community"])}, {"result", app.StringValue(response["result"])}})
		},
	}
	cmd.Flags().IntVar(&templateID, "template-id", 0, "numeric template ID")
	cmd.Flags().BoolVar(&enabled, "enabled", true, "set the community flag")
	return cmd
}