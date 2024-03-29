package main

import (
	"context"
	"fmt"
	"log"

	"os"

	"github.com/urfave/cli/v3"
)

var DEFAULT_FILE_NAME = "gitlab_variables.json"

func main() {
	cli.VersionPrinter = func(cmd *cli.Command) {
		fmt.Fprintf(cmd.Root().Writer, "%s\n", cmd.Root().Version)
	}
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	app := &cli.Command{
		Name:    "credder",
		Usage:   "manage GitLab CI variables in version control.",
		Version: "1.0.0",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "file",
				Aliases: []string{"f"},
				Value:   DEFAULT_FILE_NAME,
				Usage:   "Path to the variables file.",
			},
			&cli.StringFlag{
				Name:     "gitlab-token",
				Aliases:  []string{"g"},
				Value:    "",
				Usage:    "GitLab API token",
				Sources:  cli.EnvVars("GL_PAT", "GITLAB_TOKEN"),
				Required: true,
			},
		},
		EnableShellCompletion: true,
		Commands: []*cli.Command{
			{
				Name:    "init",
				Aliases: []string{},
				Usage:   "Set up a new variable file.",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					project_id := GetProjectID()
					init_variables(project_id)
					return nil
				},
			},
			{
				Name:    "import",
				Aliases: []string{},
				Usage:   "Overwrite local variables with remote.",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					Import()
					return nil
				},
			},
			{
				Name:    "pull",
				Aliases: []string{},
				Usage:   "Update local variables with remote.",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					Pull()
					return nil
				},
			},
			{
				Name:    "push",
				Aliases: []string{},
				Usage:   "Update remote variables with local.",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					Push()
					return nil
				},
			},
			{
				Name:    "diff",
				Aliases: []string{},
				Usage:   "Show staged local changes (what will change on GitLab).",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					Diff()
					return nil
				},
			},
			{
				Name:    "format",
				Aliases: []string{},
				Usage:   "Format the local variables file; reorders and nests.",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					local := ProjectSecrets{}
					local.Read(DEFAULT_FILE_NAME)
					local.Write(DEFAULT_FILE_NAME)
					return nil
				},
			},
		},
		// Action: func(ctx context.Context, cmd *cli.Command) error {
		// 	fmt.Println("Please specify a command.")
		// 	return nil
		// },
	}

	if err := app.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
