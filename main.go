package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os/exec"
	"sort"
	"strings"

	"github.com/creack/pty"

	"os"

	"github.com/xanzy/go-gitlab"
)

type NestedProjectSecrets struct {
	ProjectID int            `json:"project_id"`
	Variables []NestedSecret `json:"variables"`
}

type ProjectSecrets struct {
	ProjectID int      `json:"project_id"`
	Variables []Secret `json:"variables"`
}

func (project *NestedProjectSecrets) Unnest() ProjectSecrets {
	unnestedProject := ProjectSecrets{
		ProjectID: project.ProjectID,
		Variables: []Secret{},
	}

	for _, parent := range project.Variables {
		if len(parent.Nested) == 0 {
			unnestedProject.Variables = append(unnestedProject.Variables, Secret{
				Key:          parent.Key,
				Value:        *parent.Value,
				Description:  *parent.Description,
				VariableType: *parent.VariableType,
				Environment:  *parent.Environment,
				Protect:      *parent.Protect,
				Mask:         *parent.Mask,
				Raw:          *parent.Raw,
			})
		} else {
			for _, nested := range parent.Nested {
				value := parent.Value
				description := parent.Description
				vartype := parent.VariableType
				env := parent.Environment
				protect := parent.Protect
				mask := parent.Mask
				raw := parent.Raw
				if parent.Value != nil {
					value = nested.Value
				}
				if parent.Description != nil {
					description = nested.Description
				}
				if parent.VariableType != nil {
					vartype = nested.VariableType
				}
				if parent.Environment != nil {
					env = nested.Environment
				}
				if parent.Protect != nil {
					protect = nested.Protect
				}
				if parent.Mask != nil {
					mask = nested.Mask
				}
				if parent.Raw != nil {
					raw = nested.Raw
				}

				unnestedProject.Variables = append(unnestedProject.Variables, Secret{
					Key:          parent.Key,
					Value:        *value,
					Description:  *description,
					VariableType: *vartype,
					Environment:  *env,
					Protect:      *protect,
					Mask:         *mask,
					Raw:          *raw,
				})
			}
		}
	}
	sort_variables(unnestedProject)
	return unnestedProject
}

func (project *ProjectSecrets) Nest() NestedProjectSecrets {
	// Group secrets by key
	keymap := make(map[string][]NestedSecret)
	for _, secret := range project.Variables {
		nestedSecret := NestedSecret{
			Key:          secret.Key,
			Value:        &secret.Value,
			Description:  &secret.Description,
			VariableType: &secret.VariableType,
			Environment:  &secret.Environment,
			Protect:      &secret.Protect,
			Mask:         &secret.Mask,
			Raw:          &secret.Raw,
			Nested:       []NestedSecret{},
		}
		keymap[secret.Key] = append(keymap[secret.Key], nestedSecret)
	}

	// for each group, nest secrets
	topLevelGroup := []NestedSecret{}
	for key, secrets := range keymap {
		if len(secrets) == 1 {
			// no need to nest
			topLevelGroup = append(topLevelGroup, secrets[0])
			continue
		}
		parentSecret := NestedSecret{
			Key:          key,
			Value:        secrets[0].Value,
			Description:  secrets[0].Description,
			VariableType: secrets[0].VariableType,
			Environment:  secrets[0].Environment,
			Protect:      secrets[0].Protect,
			Mask:         secrets[0].Mask,
			Raw:          secrets[0].Raw,
			Nested:       []NestedSecret{},
		}
		// check for all fields if they are the same
		value := true
		description := true
		vartype := true
		env := true
		protect := true
		mask := true
		raw := true
		for _, secret := range secrets {
			if *secret.Value != *secrets[0].Value {
				parentSecret.Value = nil
				value = false
			}
			if *secret.Description != *secrets[0].Description {
				parentSecret.Description = nil
				description = false
			}
			if *secret.VariableType != *secrets[0].VariableType {
				parentSecret.VariableType = nil
				vartype = false
			}
			if *secret.Environment != *secrets[0].Environment {
				parentSecret.Environment = nil
				env = false
			}
			if *secret.Protect != *secrets[0].Protect {
				parentSecret.Protect = nil
				protect = false
			}
			if *secret.Mask != *secrets[0].Mask {
				parentSecret.Mask = nil
				mask = false
			}
			if *secret.Raw != *secrets[0].Raw {
				parentSecret.Raw = nil
				raw = false
			}
		}
		// Remove fields if they are the same
		for _, secret := range secrets {
			secret.Key = ""
			if value {
				secret.Value = nil
			}
			if description {
				secret.Description = nil
			}
			if vartype {
				secret.VariableType = nil
			}
			if env {
				secret.Environment = nil
			}
			if protect {
				secret.Protect = nil
			}
			if mask {
				secret.Mask = nil
			}
			if raw {
				secret.Raw = nil
			}
			parentSecret.Nested = append(parentSecret.Nested, secret)
		}

		topLevelGroup = append(topLevelGroup, parentSecret)
	}
	nestedProject := NestedProjectSecrets{
		ProjectID: project.ProjectID,
		Variables: topLevelGroup,
	}
	return nestedProject
}

type NestedSecret struct {
	Key          string         `json:"key,omitempty"`
	Value        *string        `json:"value,omitempty"`
	Description  *string        `json:"description,omitempty"`
	VariableType *string        `json:"type,omitempty"`
	Environment  *string        `json:"env,omitempty"`
	Protect      *bool          `json:"protect,omitempty"`
	Mask         *bool          `json:"mask,omitempty"`
	Raw          *bool          `json:"raw,omitempty"`
	Nested       []NestedSecret `json:"nested,omitempty"`
}

func (nestedProject *NestedProjectSecrets) save_local() error {
	content, err := json.MarshalIndent(nestedProject, "", "  ")
	if err != nil {
		fmt.Println("Error encoding JSON:", err)
		return err
	}
	err = os.WriteFile("gitlab_nested_variables.json", content, 0644)
	if err != nil {
		fmt.Println("Could not write variables file:", err)
		return err
	}
	return nil
}

type Secret struct {
	Key          string `json:"key"`
	Value        string `json:"value"`
	Description  string `json:"description"`
	VariableType string `json:"type"`
	Environment  string `json:"env"`
	Protect      bool   `json:"protect"`
	Mask         bool   `json:"mask"`
	Raw          bool   `json:"raw"`
}

func (secret *Secret) UnmarshalJSON(data []byte) error {
	secret.Raw = true
	secret.Mask = false
	secret.Protect = true
	secret.Environment = "*"

	type tempSecret Secret
	return json.Unmarshal(data, (*tempSecret)(secret))
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	// log.SetFlags(log.LstdFlags | log.Lshortfile)

	var command string

	if len(os.Args) >= 2 {
		command = os.Args[1]
	} else {
		command = ""
	}

	switch command {
	case "init":
		project_id := getProjectID()
		// if len(os.Args) < 3 {
		// 	log.Fatal("Missing project ID")
		// }
		// projectID, err := strconv.Atoi(os.Args[2])
		// if err != nil {
		// 	log.Fatal("Invalid project ID (not a number)")
		// }
		init_variables(project_id)
	case "import":
		import_gl()
	case "pull":
		pull()
	case "push":
		push()
	case "diff":
		diff()
	case "nest":
		local, err := get_local_variables()
		if err != nil {
			fmt.Println("Could not load local variables file:", err)
			return
		}
		nested := local.Nest()
		nested.save_local()

	default:
		fmt.Println(`Usage: gitlab-secrets [init|import|pull|push|diff|help] [OPTIONS]
Init: Pull using a project ID.
Pull: update the local file with the secrets from GitLab. Values will be unset.
Push: update the GitLab secrets with the local file. Will inject passwords.
Diff: show the difference between local and remote secrets.
Help: This message.`)

	}
}

func getProjectID() int {
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

func showDiff(a string, b string) {
	cmd := exec.Command("bash", "-c", fmt.Sprintf("diff -U 8 --color=always <(echo '%s') <(echo '%s') && echo 'no diff'", a, b))
	f, err := pty.Start(cmd)
	io.Copy(os.Stderr, f)
	exit_code := cmd.ProcessState.ExitCode()
	if err != nil {
		fmt.Println(exit_code)
		fmt.Println("Error executing diff command:", err)
		return
	}

}

func get_gitlab_client() *gitlab.Client {
	git, err := gitlab.NewClient(os.Getenv("GL_PAT"))
	if err != nil {
		log.Fatal("Could not create GitLab client:", err)
	}
	return git
}

func get_remote_variables(project_id int) ([]*gitlab.ProjectVariable, error) {
	git := get_gitlab_client()

	variables, _, err := git.ProjectVariables.ListVariables(project_id, &gitlab.ListProjectVariablesOptions{
		PerPage: 100,
	})
	if err != nil {
		return nil, err
	}
	return variables, nil
}

func get_local_variables() (ProjectSecrets, error) {
	byteValue, err := os.ReadFile("gitlab_variables.json")
	if err != nil {
		fmt.Println("Error reading file:", err)
		return ProjectSecrets{}, err
	}

	var project ProjectSecrets

	err = json.Unmarshal(byteValue, &project)
	if err != nil {
		fmt.Println("Error decoding JSON:", err)
		return ProjectSecrets{}, err
	}

	return project, nil
}

func sort_variables(project ProjectSecrets) {
	sort.Slice(project.Variables, func(i, j int) bool {
		if project.Variables[i].Key != project.Variables[j].Key {
			return project.Variables[i].Key < project.Variables[j].Key
		}
		return project.Variables[i].Environment < project.Variables[j].Environment
	})
}

func remote_to_local(project_id int, remote []*gitlab.ProjectVariable) ProjectSecrets {
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

func extractFileVariables(project ProjectSecrets) []Secret {
	fileVariables := []Secret{}
	for _, variable := range project.Variables {
		if variable.VariableType == "file" {
			fileVariables = append(fileVariables, variable)
		}
	}
	return fileVariables
}

type ValueTuple struct {
	Local  string
	Remote string
}

func diff() {
	local, err := get_local_variables()
	if err != nil {
		fmt.Println("Could not load local variables file:", err)
		return
	}
	local = inject_secrets(inject_files(local))
	gl_remote, err := get_remote_variables(local.ProjectID)
	if err != nil {
		fmt.Println("Could not load remote variables:", err)
		return
	}
	remote := remote_to_local(local.ProjectID, gl_remote)

	sort_variables(local)
	sort_variables(remote)

	// marshall to json with indents
	localJson, err := json.MarshalIndent(local, "", "  ")
	if err != nil {
		fmt.Println("Error encoding JSON:", err)
		return
	}
	remoteJson, err := json.MarshalIndent(remote, "", "  ")
	if err != nil {
		fmt.Println("Error encoding JSON:", err)
		return
	}

	// call command line funcion diff
	showDiff(string(remoteJson), string(localJson))

	// Calculate file diffs
	localFileVariables := extractFileVariables(local)
	remoteFileVariables := extractFileVariables(remote)

	fileTuples := make(map[KeyTuple]ValueTuple)
	for _, secret := range localFileVariables {
		fileTuples[KeyTuple{
			Key:         secret.Key,
			Environment: secret.Environment,
		}] = ValueTuple{
			Local:  secret.Value,
			Remote: "",
		}
	}

	for _, secret := range remoteFileVariables {
		fileTuples[KeyTuple{
			Key:         secret.Key,
			Environment: secret.Environment,
		}] = ValueTuple{
			Local:  fileTuples[KeyTuple{Key: secret.Key, Environment: secret.Environment}].Local,
			Remote: secret.Value,
		}
	}

	for key, value := range fileTuples {
		if value.Local != value.Remote {
			fmt.Printf("====== %s (%s) ======\n", key.Key, key.Environment)
			showDiff(value.Remote, value.Local)
		}
	}
}

func save_local(project ProjectSecrets) error {
	content, err := json.MarshalIndent(project, "", "  ")
	if err != nil {
		fmt.Println("Error encoding JSON:", err)
		return err
	}
	err = os.WriteFile("gitlab_variables.json", content, 0644)
	if err != nil {
		fmt.Println("Could not write variables file:", err)
		return err
	}
	return nil
}

func init_variables(project_id int) {
	_, err := os.Stat("gitlab_variables.json")
	if !errors.Is(err, os.ErrNotExist) {
		fmt.Println("Variables file (gitlab_variables.json) already exists, or something else went wrong:", err)
		return
	}
	var project ProjectSecrets = ProjectSecrets{
		ProjectID: project_id,
		Variables: []Secret{},
	}

	content, err := json.Marshal(project)
	if err != nil {
		fmt.Println("Error encoding JSON:", err)
		return
	}
	err = os.WriteFile("gitlab_variables.json", content, 0644)
	if err != nil {
		fmt.Println("Could not write variables file:", err)
		return
	}
	pull()
}

func inject_files(project ProjectSecrets) ProjectSecrets {
	newProject := ProjectSecrets{
		ProjectID: project.ProjectID,
		Variables: make([]Secret, len(project.Variables)),
	}

	for i, secret := range project.Variables {
		newProject.Variables[i] = Secret{
			Key:          secret.Key,
			Value:        secret.Value,
			Description:  secret.Description,
			VariableType: secret.VariableType,
			Environment:  secret.Environment,
			Protect:      secret.Protect,
			Mask:         secret.Mask,
			Raw:          secret.Raw,
		}
	}

	for i := range newProject.Variables {
		if newProject.Variables[i].VariableType == "file" {
			if newProject.Variables[i].Value == "" {
				continue
			}
			fileContent, err := os.ReadFile(newProject.Variables[i].Value)
			if err != nil {
				fmt.Println("Error loading file:", err)
				continue
			}
			newProject.Variables[i].Value = string(fileContent)
		}
	}
	return newProject
}

func inject_secrets(project ProjectSecrets) ProjectSecrets {
	// marshall to json with indents
	localJson, err := json.MarshalIndent(project, "", "  ")
	if err != nil {
		fmt.Println("Error encoding JSON:", err)
		return project
	}
	cmd := exec.Command("bash", "-c", fmt.Sprintf("echo '%s' | op inject", localJson))
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println("Error injecting secrets:", err)
		fmt.Println(string(output))
		log.Fatalln("Could not inject secrets")
		return project
	}
	var injected ProjectSecrets
	err = json.Unmarshal(output, &injected)
	if err != nil {
		fmt.Println("Error decoding JSON:", err)
		return project
	}

	return injected
}

type KeyTuple struct {
	Key         string
	Environment string
}

func pull() {
	local, err := get_local_variables()
	if err != nil {
		fmt.Println("Could not load local variables file:", err)
		return
	}
	sort_variables(local)

	injected := inject_secrets(inject_files(local))

	remote_vars, err := get_remote_variables(local.ProjectID)
	if err != nil {
		fmt.Println("Could not load remote variables:", err)
		return
	}
	r2l := remote_to_local(local.ProjectID, remote_vars)
	sort_variables(r2l)
	remote := r2l.Variables

	// map local keys to local values
	localMap := make(map[KeyTuple]string)
	for _, secret := range local.Variables {
		localMap[KeyTuple{
			Key:         secret.Key,
			Environment: secret.Environment,
		}] = secret.Value
	}
	injectedMap := make(map[KeyTuple]string)
	for _, secret := range injected.Variables {
		injectedMap[KeyTuple{
			Key:         secret.Key,
			Environment: secret.Environment,
		}] = secret.Value
	}

	local.Variables = []Secret{}
	for i := 0; i < len(remote); i++ {
		value := ""
		if injectedMap[KeyTuple{
			Key:         remote[i].Key,
			Environment: remote[i].Environment,
		}] == remote[i].Value {
			value = localMap[KeyTuple{
				Key:         remote[i].Key,
				Environment: remote[i].Environment,
			}]
		}
		local.Variables = append(local.Variables, Secret{
			Key: remote[i].Key,
			// Set the value to the local value, or empty if not found
			Value:        value,
			Description:  remote[i].Description,
			VariableType: string(remote[i].VariableType),
			Environment:  remote[i].Environment,
			Protect:      remote[i].Protect,
			Mask:         remote[i].Mask,
			Raw:          remote[i].Raw,
		})
	}

	err = save_local(local)
	if err != nil {
		fmt.Println("Could not save local variables file:", err)
		return
	}
}

func import_gl() {
	fmt.Println("Importing GitLab variables to local file, DO NOT PUSH TO REPO; CONTAINS SECRETS")
	project_id := getProjectID()
	remote, err := get_remote_variables(project_id)
	if err != nil {
		fmt.Println("Could not load remote variables:", err)
		return
	}
	local := ProjectSecrets{
		ProjectID: project_id,
		Variables: []Secret{},
	}

	for i := 0; i < len(remote); i++ {
		value := remote[i].Value
		if remote[i].VariableType == "file" {
			// create directory if it does not exist
			if _, err := os.Stat("variables"); os.IsNotExist(err) {
				err := os.Mkdir("variables", 0755)
				if err != nil {
					fmt.Println("Could not create directory:", err)
					return
				}
			}

			// store the variable with secrets
			value = "./variables/" + remote[i].Key + "_" + base64.StdEncoding.EncodeToString([]byte(remote[i].EnvironmentScope))
			err := os.WriteFile(value, []byte(remote[i].Value), 0644)
			if err != nil {
				fmt.Println("Could not write file:", err)
			}
		}

		local.Variables = append(local.Variables, Secret{
			Key: remote[i].Key,
			// Set the value to the local value, or empty if not found
			Value:        value,
			Description:  remote[i].Description,
			VariableType: string(remote[i].VariableType),
			Environment:  remote[i].EnvironmentScope,
			Protect:      remote[i].Protected,
			Mask:         remote[i].Masked,
			Raw:          remote[i].Raw,
		})
	}

	err = save_local(local)
	if err != nil {
		fmt.Println("Could not save local variables file:", err)
		return
	}
}

func push() {
	// Read the file
	local, err := get_local_variables()
	if err != nil {
		fmt.Println("Could not load local variables file:", err)
		return
	}

	local = inject_secrets(inject_files(local))

	// Get the remote variables
	remote, err := get_remote_variables(local.ProjectID)
	if err != nil {
		fmt.Println("Could not load remote variables:", err)
		return
	}

	// Find variables only in local and overlapping ones
	localOnly := []Secret{}
	overlappingLocal := []Secret{}
	overlappingRemote := []Secret{}
	for _, localVar := range local.Variables {
		found := false
		for _, remoteVar := range remote {
			if localVar.Key == remoteVar.Key && localVar.Environment == remoteVar.EnvironmentScope {
				overlappingLocal = append(overlappingLocal, localVar)
				overlappingRemote = append(overlappingRemote, Secret{
					Key:          remoteVar.Key,
					Value:        remoteVar.Value,
					Description:  remoteVar.Description,
					VariableType: string(remoteVar.VariableType),
					Environment:  remoteVar.EnvironmentScope,
					Protect:      remoteVar.Protected,
					Mask:         remoteVar.Masked,
					Raw:          remoteVar.Raw,
				})
				found = true
				break
			}
		}
		if !found {
			localOnly = append(localOnly, localVar)
		}
	}

	// Find variables only in remote
	remoteOnly := []*gitlab.ProjectVariable{}
	for _, remoteVar := range remote {
		found := false
		for _, localVar := range local.Variables {
			if localVar.Key == remoteVar.Key && localVar.Environment == remoteVar.EnvironmentScope {
				found = true
				break
			}
		}
		if !found {
			remoteOnly = append(remoteOnly, remoteVar)
		}
	}

	git := get_gitlab_client()

	// CREATION:
	var input string
	// create local only remote
	for _, localVar := range localOnly {
		fmt.Println("Creating variable:", localVar.Key, localVar.Environment)
		jsonVar, err := json.MarshalIndent(localVar, "", "  ")
		if err == nil {
			fmt.Println(string(jsonVar))
		} else {
			fmt.Println("ERROR")
		}
		fmt.Println("CREATE? (y/n): ")
		fmt.Scanln(&input)
		if input != "y" {
			continue
		}
		variableType := gitlab.VariableTypeValue(localVar.VariableType)
		_, _, err = git.ProjectVariables.CreateVariable(local.ProjectID, &gitlab.CreateProjectVariableOptions{
			Key:              &localVar.Key,
			Value:            &localVar.Value,
			Description:      &localVar.Description,
			EnvironmentScope: &localVar.Environment,
			Masked:           &localVar.Mask,
			Raw:              &localVar.Raw,
			Protected:        &localVar.Protect,
			VariableType:     &variableType,
		})
		if err != nil {
			fmt.Println("Could not create variable:", err)
		}
	}

	// update overlapping variables
	for i, localVar := range overlappingLocal {
		if localVar == overlappingRemote[i] {
			continue
		}
		fmt.Println("Updating variable:", localVar.Key, localVar.Environment)
		remoteSecret := overlappingRemote[i]
		jsonLocalVar, err := json.MarshalIndent(localVar, "", "  ")
		jsonRemoteVar, err2 := json.MarshalIndent(remoteSecret, "", "  ")
		if err == nil && err2 == nil {
			showDiff(string(jsonRemoteVar), string(jsonLocalVar))
			if localVar.VariableType == "file" && localVar.Value != remoteSecret.Value {
				showDiff(remoteSecret.Value, localVar.Value)
			}
		} else {
			fmt.Println("ERROR")
		}
		fmt.Println("Do you want to UPDATE this variable? (y/n): ")
		fmt.Scanln(&input)
		if input != "y" {
			continue
		}
		variableType := gitlab.VariableTypeValue(localVar.VariableType)
		_, _, err = git.ProjectVariables.UpdateVariable(local.ProjectID, localVar.Key, &gitlab.UpdateProjectVariableOptions{
			Value:            &localVar.Value,
			Description:      &localVar.Description,
			EnvironmentScope: &localVar.Environment,
			Masked:           &localVar.Mask,
			Raw:              &localVar.Raw,
			Protected:        &localVar.Protect,
			VariableType:     &variableType,
			Filter: &gitlab.VariableFilter{
				EnvironmentScope: localVar.Environment,
			},
		})
		if err != nil {
			fmt.Println("Could not update variable:", err)
		}
	}

	// Delete variables
	for _, remoteVar := range remoteOnly {
		fmt.Println("Deleting variable:", remoteVar.Key, remoteVar.EnvironmentScope)
		fmt.Println(remoteVar)
		fmt.Println("Do you want to DELETE this variable? (y/n): ")
		fmt.Scanln(&input)
		if input != "y" {
			continue
		}
		_, err := git.ProjectVariables.RemoveVariable(local.ProjectID, remoteVar.Key, &gitlab.RemoveProjectVariableOptions{
			Filter: &gitlab.VariableFilter{
				EnvironmentScope: remoteVar.EnvironmentScope,
			},
		})
		if err != nil {
			fmt.Println("Could not delete variable:", err)
		}
	}

}
