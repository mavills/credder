package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

func init_variables(project_id int) {
	_, err := os.Stat(DEFAULT_FILE_NAME)
	if !errors.Is(err, os.ErrNotExist) {
		fmt.Printf("Variables file (%s) already exists, or something else went wrong: %s", DEFAULT_FILE_NAME, err)
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
	err = os.WriteFile(DEFAULT_FILE_NAME, content, 0644)
	if err != nil {
		fmt.Println("Could not write variables file:", err)
		return
	}
	Pull()
}
