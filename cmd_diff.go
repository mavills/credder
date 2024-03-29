package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/creack/pty"
)

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

func Diff() {
	local := ProjectSecrets{}
	remote := ProjectSecrets{}

	err := local.Read(DEFAULT_FILE_NAME)
	if err != nil {
		fmt.Println("Could not load local variables file:", err)
		return
	}
	local = local.InjectFiles().InjectSecrets()

	err = remote.FetchVariables(local.ProjectID)
	if err != nil {
		fmt.Println("Could not load remote variables:", err)
		return
	}

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
	localFileVariables := local.FileVariables()
	remoteFileVariables := remote.FileVariables()

	type KeyTuple struct {
		Key         string
		Environment string
	}
	type ValueTuple struct {
		Local  string
		Remote string
	}

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
