package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"github.com/xanzy/go-gitlab"
)

func GetProjectID() int {
	cmd := exec.Command("git", "remote", "get-url", "origin")
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatalln("Could not get project id:", err)
	}
	a := string(output)
	a = strings.Split(a, ":")[1]
	a = strings.Split(a, ".git")[0]
	a = strings.Replace(a, "/", "%2F", -1)

	token := os.Getenv("GL_PAT")
	url := fmt.Sprintf("https://gitlab.com/api/v4/projects/%s", a)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatal("Could not create HTTP request:", err)
	}
	req.Header.Set("PRIVATE-TOKEN", token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatal("Could not make request to GitLab API:", err)
	}
	defer resp.Body.Close()

	var data map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&data)
	if err != nil {
		log.Fatal("Could not decode JSON response:", err)
	}

	project_id := data["id"].(float64)

	id := int(project_id)
	return id
}

func getGitlabClient() *gitlab.Client {
	git, err := gitlab.NewClient(os.Getenv("GL_PAT"))
	if err != nil {
		log.Fatal("Could not create GitLab client:", err)
	}
	return git
}

func remoteToLocal(project_id int, remote []*gitlab.ProjectVariable) ProjectSecrets {
	local := ProjectSecrets{
		ProjectID: project_id,
		Variables: []Secret{},
	}
	for i := 0; i < len(remote); i++ {
		local.Variables = append(local.Variables, Secret{
			Key:          remote[i].Key,
			Value:        remote[i].Value,
			Description:  remote[i].Description,
			VariableType: string(remote[i].VariableType),
			Environment:  remote[i].EnvironmentScope,
			Protect:      remote[i].Protected,
			Mask:         remote[i].Masked,
			Raw:          remote[i].Raw,
		})
	}
	return local
}

// FetchVariables fetches the project variables for a given project ID from GitLab.
// It returns a struct of type ProjectSecrets that contains the fetched variables,
// or an error if the fetching process fails.
func (project ProjectSecrets) FetchVariables(project_id int) error {
	git := getGitlabClient()

	var variables []*gitlab.ProjectVariable
	page := 1
	perPage := 100

	for {
		opts := &gitlab.ListProjectVariablesOptions{
			Page:    page,
			PerPage: perPage,
		}

		vars, resp, err := git.ProjectVariables.ListVariables(project_id, opts)
		if err != nil {
			return err
		}

		variables = append(variables, vars...)

		if resp.CurrentPage >= resp.TotalPages {
			break
		}

		page = resp.NextPage
	}

	filled := remoteToLocal(project_id, variables)
	project.ProjectID = project_id
	project.Variables = filled.Variables
	project.Order()

	return nil
}

// CreateVariable creates a new variable for a project in GitLab.
// It takes the project ID and a Secret variable as parameters.
// The function returns an error if the variable creation fails.
func CreateVariable(projectId int, variable Secret) error {
	git := getGitlabClient()

	variableType := gitlab.VariableTypeValue(variable.VariableType)
	_, _, err := git.ProjectVariables.CreateVariable(projectId, &gitlab.CreateProjectVariableOptions{
		Value:            &variable.Value,
		Description:      &variable.Description,
		EnvironmentScope: &variable.Environment,
		Masked:           &variable.Mask,
		Raw:              &variable.Raw,
		Protected:        &variable.Protect,
		VariableType:     &variableType,
	})
	return err
}

// UpdateVariable updates a variable for a given project in GitLab.
// It takes the project ID and a Secret variable as parameters.
// Returns an error if the update operation fails.
func UpdateVariable(projectId int, variable Secret) error {
	git := getGitlabClient()

	variableType := gitlab.VariableTypeValue(variable.VariableType)
	_, _, err := git.ProjectVariables.UpdateVariable(projectId, variable.Key, &gitlab.UpdateProjectVariableOptions{
		Value:            &variable.Value,
		Description:      &variable.Description,
		EnvironmentScope: &variable.Environment,
		Masked:           &variable.Mask,
		Raw:              &variable.Raw,
		Protected:        &variable.Protect,
		VariableType:     &variableType,
		Filter: &gitlab.VariableFilter{
			EnvironmentScope: variable.Environment,
		},
	})
	return err
}

// DeleteVariable removes a variable from a project in GitLab.
// It takes the project ID, variable key, and environment scope as parameters.
// Returns an error if the variable deletion fails.
func DeleteVariable(projectId int, key string, environment string) error {
	git := getGitlabClient()
	_, err := git.ProjectVariables.RemoveVariable(projectId, key, &gitlab.RemoveProjectVariableOptions{
		Filter: &gitlab.VariableFilter{
			EnvironmentScope: environment,
		},
	})
	return err
}
