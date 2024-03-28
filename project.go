package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
)

type NestedProjectSecrets struct {
	ProjectID int            `json:"project_id"`
	Variables []NestedSecret `json:"variables"`
}

func (nestedProject *NestedProjectSecrets) Write(filename string) error {
	content, err := json.MarshalIndent(nestedProject, "", "  ")
	if err != nil {
		fmt.Println("Error encoding JSON:", err)
		return err
	}
	err = os.WriteFile(filename, content, 0644)
	if err != nil {
		fmt.Println("Could not write variables file:", err)
		return err
	}
	return nil
}

func (nestedProject *NestedProjectSecrets) Read(filename string) error {
	byteValue, err := os.ReadFile(filename)
	if err != nil {
		fmt.Println("Error reading file:", err)
		return err
	}

	err = json.Unmarshal(byteValue, &nestedProject)
	if err != nil {
		fmt.Println("Error decoding JSON:", err)
		return err
	}
	return nil
}

func (nestedProject NestedProjectSecrets) Equal(other NestedProjectSecrets) bool {
	if nestedProject.ProjectID != other.ProjectID {
		return false
	}
	if len(nestedProject.Variables) != len(other.Variables) {
		return false
	}
	for i, variable := range nestedProject.Variables {
		if !variable.Equal(other.Variables[i]) {
			return false
		}
	}
	return true
}

type ProjectSecrets struct {
	ProjectID int      `json:"project_id"`
	Variables []Secret `json:"variables"`
}

func (project *ProjectSecrets) Order() {
	sort.Slice(project.Variables, func(i, j int) bool {
		if project.Variables[i].Key != project.Variables[j].Key {
			return project.Variables[i].Key < project.Variables[j].Key
		}
		return project.Variables[i].Environment < project.Variables[j].Environment
	})
}

func (project *ProjectSecrets) Read(filename string) error {
	nestedProject := NestedProjectSecrets{}
	err := nestedProject.Read(filename)
	if err != nil {
		return err
	}
	unnested := nestedProject.Unnest()
	project.ProjectID = unnested.ProjectID
	project.Variables = unnested.Variables
	project.Order()
	return nil
}

func (project *ProjectSecrets) Write(filename string) error {
	project.Order()
	nested := project.Nest()
	err := nested.Write(filename)
	if err != nil {
		return err
	}
	return nil
}

func (project ProjectSecrets) FileVariables() []Secret {
	fileVariables := []Secret{}
	for _, variable := range project.Variables {
		if variable.VariableType == "file" {
			fileVariables = append(fileVariables, variable)
		}
	}
	return fileVariables
}

func (project ProjectSecrets) Equal(other ProjectSecrets) bool {
	if project.ProjectID != other.ProjectID {
		return false
	}
	if len(project.Variables) != len(other.Variables) {
		return false
	}
	for i, variable := range project.Variables {
		if !variable.Equal(other.Variables[i]) {
			return false
		}
	}
	return true
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

func (secret *NestedSecret) Equal(other NestedSecret) bool {
	if secret.Key != other.Key {
		fmt.Printf("Key: %s != %s\n", secret.Key, other.Key)
		return false
	}
	if secret.Value != other.Value && *secret.Value != *other.Value {
		fmt.Printf("Value: %s != %s\n", *secret.Value, *other.Value)
		return false
	}
	if (secret.Description != other.Description) && *secret.Description != *other.Description {
		fmt.Printf("Description: %s != %s\n", *secret.Description, *other.Description)
		return false
	}
	if secret.VariableType != other.VariableType && *secret.VariableType != *other.VariableType {
		fmt.Printf("Type %s != %s\n", *secret.VariableType, *other.VariableType)
		return false
	}
	if secret.Environment != other.Environment && *secret.Environment != *other.Environment {
		fmt.Printf("Environment: %s != %s\n", *secret.Environment, *other.Environment)
		return false
	}
	if secret.Environment != other.Environment && *secret.Protect != *other.Protect {
		fmt.Printf("Protect: %t != %t\n", *secret.Protect, *other.Protect)
		return false
	}
	if secret.Mask != other.Mask && *secret.Mask != *other.Mask {
		fmt.Printf("Mask: %t != %t\n", *secret.Mask, *other.Mask)
		return false
	}
	if secret.Raw != other.Raw && *secret.Raw != *other.Raw {
		fmt.Printf("Raw: %t != %t\n", *secret.Raw, *other.Raw)
		return false
	}
	if len(secret.Nested) != len(other.Nested) {
		fmt.Printf("Nested: %d != %d\n", len(secret.Nested), len(other.Nested))
		return false
	}
	for i, nested := range secret.Nested {
		if !nested.Equal(other.Nested[i]) {
			return false
		}
	}
	return true
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

func (secret *Secret) Equal(other Secret) bool {
	if secret.Key != other.Key {
		return false
	}
	if secret.Value != other.Value {
		return false
	}
	if secret.Description != other.Description {
		return false
	}
	if secret.VariableType != other.VariableType {
		return false
	}
	if secret.Environment != other.Environment {
		return false
	}
	if secret.Protect != other.Protect {
		return false
	}
	if secret.Mask != other.Mask {
		return false
	}
	if secret.Raw != other.Raw {
		return false
	}
	return true
}
