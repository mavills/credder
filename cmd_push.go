package main

import (
	"encoding/json"
	"fmt"
)

func Push() {
	// Read the file
	local := ProjectSecrets{}
	remote := ProjectSecrets{}
	err := local.Read(DEFAULT_FILE_NAME)
	if err != nil {
		fmt.Println("Could not load local variables file:", err)
		return
	}
	local = local.InjectFiles().InjectSecrets()

	// Get the remote variables
	err = remote.FetchVariables(local.ProjectID)
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
		for _, remoteVar := range remote.Variables {
			if localVar.Key == remoteVar.Key && localVar.Environment == remoteVar.Environment {
				overlappingLocal = append(overlappingLocal, localVar)
				overlappingRemote = append(overlappingRemote, remoteVar)
				found = true
				break
			}
		}
		if !found {
			localOnly = append(localOnly, localVar)
		}
	}

	// Find variables only in remote
	remoteOnly := []Secret{}
	for _, remoteVar := range remote.Variables {
		found := false
		for _, localVar := range local.Variables {
			if localVar.Key == remoteVar.Key && localVar.Environment == remoteVar.Environment {
				found = true
				break
			}
		}
		if !found {
			remoteOnly = append(remoteOnly, remoteVar)
		}
	}

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
		err = CreateVariable(local.ProjectID, localVar)
		if err != nil {
			fmt.Println("Could not CREATE variable:", err)
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
		err = UpdateVariable(local.ProjectID, localVar)
		if err != nil {
			fmt.Println("Could not update variable:", err)
		}
	}

	// Delete variables
	for _, remoteVar := range remoteOnly {
		fmt.Println("Deleting variable:", remoteVar.Key, remoteVar.Environment)
		fmt.Println(remoteVar)
		fmt.Println("Do you want to DELETE this variable? (y/n): ")
		fmt.Scanln(&input)
		if input != "y" {
			continue
		}
		err = DeleteVariable(local.ProjectID, remoteVar.Key, remoteVar.Environment)
		if err != nil {
			fmt.Println("Could not delete variable:", err)
		}
	}

}
