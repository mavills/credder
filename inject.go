package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
)

// Return a copy of the project with filenames injected
func (project ProjectSecrets) InjectFiles() ProjectSecrets {
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

// Return a copy of the project with secrets injected
func (project ProjectSecrets) InjectSecrets() ProjectSecrets {
	// marshall to json with indents
	localJson, err := json.MarshalIndent(project, "", "  ")
	if err != nil {
		fmt.Println("Error encoding JSON:", err)
		return ProjectSecrets{}
	}
	cmd := exec.Command("bash", "-c", fmt.Sprintf("echo '%s' | op inject", localJson))
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println("Error injecting secrets:", err)
		fmt.Println(string(output))
		log.Fatalln("Could not inject secrets")
		return ProjectSecrets{}
	}
	var injected ProjectSecrets
	err = json.Unmarshal(output, &injected)
	if err != nil {
		fmt.Println("Error decoding JSON:", err)
		return ProjectSecrets{}
	}

	return injected
}
