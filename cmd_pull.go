package main

import "fmt"

func Pull() {
	local := ProjectSecrets{}
	remote := ProjectSecrets{}

	err := local.Read(DEFAULT_FILE_NAME)
	if err != nil {
		fmt.Println("Could not load local variables file:", err)
		return
	}
	injected := local.InjectFiles().InjectSecrets()

	err = remote.FetchVariables(local.ProjectID)
	if err != nil {
		fmt.Println("Could not load remote variables:", err)
		return
	}

	type KeyTuple struct {
		Key         string
		Environment string
	}
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
	for i := 0; i < len(remote.Variables); i++ {
		value := ""
		if injectedMap[KeyTuple{
			Key:         remote.Variables[i].Key,
			Environment: remote.Variables[i].Environment,
		}] == remote.Variables[i].Value {
			value = localMap[KeyTuple{
				Key:         remote.Variables[i].Key,
				Environment: remote.Variables[i].Environment,
			}]
		}
		local.Variables = append(local.Variables, Secret{
			Key: remote.Variables[i].Key,
			// Set the value to the local value, or empty if not found
			Value:        value,
			Description:  remote.Variables[i].Description,
			VariableType: string(remote.Variables[i].VariableType),
			Environment:  remote.Variables[i].Environment,
			Protect:      remote.Variables[i].Protect,
			Mask:         remote.Variables[i].Mask,
			Raw:          remote.Variables[i].Raw,
		})
	}

	err = local.Write(DEFAULT_FILE_NAME)
	if err != nil {
		fmt.Println("Could not save local variables file:", err)
		return
	}
}
