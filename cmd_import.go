package main

import (
	"encoding/base64"
	"fmt"
	"os"
)

func Import() {
	fmt.Println("Importing GitLab variables to local file, DO NOT PUSH TO REPO; CONTAINS SECRETS")
	projectId := GetProjectID()
	remote := ProjectSecrets{}
	err := remote.FetchVariables(projectId)
	if err != nil {
		fmt.Println("Could not load remote variables:", err)
		return
	}

	for i, secret := range remote.Variables {
		if secret.VariableType != "file" {
			continue
		}
		// create directory if it does not exist
		if _, err := os.Stat("variables"); os.IsNotExist(err) {
			err := os.Mkdir("variables", 0755)
			if err != nil {
				fmt.Println("Could not create directory:", err)
				return
			}
		}
		// store the variable with secrets
		path := "./variables/" + secret.Key + "_" + base64.StdEncoding.EncodeToString([]byte(secret.Environment))
		err := os.WriteFile(path, []byte(secret.Value), 0644)
		remote.Variables[i].Value = path
		if err != nil {
			fmt.Println("Could not write file:", err)
		}
	}
	err = remote.Write(DEFAULT_FILE_NAME)
	if err != nil {
		fmt.Println("Could not save local variables file:", err)
		return
	}
}
