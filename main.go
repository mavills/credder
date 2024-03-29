package main

import (
	"fmt"
	"log"

	"os"

	"github.com/urfave/cli/v2"
)

var DEFAULT_FILE_NAME = "gitlab_variables.json"

func main() {
	cli.VersionPrinter = func(cCtx *cli.Context) {
		fmt.Printf("%s\n", cCtx.App.Version)
	}
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	app := &cli.App{
		Name:    "credder",
		Usage:   "manage GitLab CI variables in version control.",
		Version: "1.0.0",
		Flags: []cli.Flag{
			// &cli.PathFlag{
			// 	Name:    "file",
			// 	Aliases: []string{"f"},
			// 	Value:   DEFAULT_FILE_NAME,
			// 	Usage:   "Path to the variables file.",
			// },
			&cli.StringFlag{
				Name:     "gitlab-token",
				Aliases:  []string{"g"},
				Value:    "",
				Usage:    "GitLab API token",
				EnvVars:  []string{"GL_PAT", "GITLAB_TOKEN"},
				Required: true,
			},
		},
		Commands: []*cli.Command{
			{
				Name:    "init",
				Aliases: []string{},
				Usage:   "Set up a new variable file.",
				Action: func(c *cli.Context) error {
					project_id := GetProjectID()
					init_variables(project_id)
					return nil
				},
			},
			{
				Name:    "import",
				Aliases: []string{},
				Usage:   "Overwrite local variables with remote.",
				Action: func(c *cli.Context) error {
					Import()
					return nil
				},
			},
			{
				Name:    "pull",
				Aliases: []string{},
				Usage:   "Update local variables with remote.",
				Action: func(c *cli.Context) error {
					Pull()
					return nil
				},
			},
			{
				Name:    "push",
				Aliases: []string{},
				Usage:   "Update remote variables with local.",
				Action: func(c *cli.Context) error {
					Push()
					return nil
				},
			},
			{
				Name:    "diff",
				Aliases: []string{},
				Usage:   "Show staged local changes (what will change on GitLab).",
				Action: func(c *cli.Context) error {
					Diff()
					return nil
				},
			},
			{
				Name:    "format",
				Aliases: []string{},
				Usage:   "Format the local variables file; reorders and nests.",
				Action: func(c *cli.Context) error {
					local := ProjectSecrets{}
					local.Read(DEFAULT_FILE_NAME)
					local.Write(DEFAULT_FILE_NAME)
					return nil
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
