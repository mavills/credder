package main

import (
	"log"

	"github.com/xanzy/go-gitlab"
)

func GetDeployTokens(project_id int) []*gitlab.DeployToken {
	git := getGitlabClient()
	deployTokens, _, err := git.DeployTokens.ListProjectDeployTokens(project_id, nil)
	if err != nil {
		log.Fatal("Could not list deploy tokens:", err)
	}
	return deployTokens
}

func CreateDeployToken(project_id int, name string) *gitlab.DeployToken {
	git := getGitlabClient()
	deployToken, _, err := git.DeployTokens.CreateProjectDeployToken(
		project_id, &gitlab.CreateProjectDeployTokenOptions{
			Name: &name,
			Scopes: &[]string{
				"read_registry",
			},
		})
	if err != nil {
		log.Fatal("Could not create deploy token:", err)
	}
	return deployToken
}
