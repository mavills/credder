package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os/exec"
	"strings"

	"github.com/creack/pty"

	"os"
)

var DEFAULT_FILE_NAME = "gitlab_variables.json"

const (
	Init     = "init"
	Import   = "import"
	Pull     = "pull"
	Diff     = "plan"
	Push     = "apply"
	Reformat = "reformat"
	Help     = "help"
)

func GetHelpString() string {
	commands := strings.Join([]string{Init, Import, Pull, Push, Diff, Reformat, Help}, "|")
	usage := fmt.Sprintf("Usage: gitlab-secrets [%s]", commands)
	init := fmt.Sprintf("\t%s: Set up a new variable file.", Init)
	imp := fmt.Sprintf("\t%s: Overwrite local variables with remote.", Import)
	pull := fmt.Sprintf("\t%s: Update local variables with remote.", Pull)
	push := fmt.Sprintf("\t%s: Update remote variables with local.", Push)
	diff := fmt.Sprintf("\t%s: Show staged local changes (what will change on GitLab).", Diff)
	reformat := fmt.Sprintf("\t%s: Reformat the local variables file.", Reformat)
	help := fmt.Sprintf("\t%s: Show this message.", Help)
	return strings.Join([]string{usage, init, imp, pull, push, diff, reformat, help}, "\n")
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	var command string

	if len(os.Args) >= 2 {
		command = os.Args[1]
	} else {
		command = ""
	}

	switch command {
	case Init:
		project_id := GetProjectID()
		init_variables(project_id)
	case Import:
		import_gl()
	case Pull:
		pull()
	case Push:
		push()
	case Diff:
		diff()
	case Reformat:
		local := ProjectSecrets{}
		local.Read(DEFAULT_FILE_NAME)
		local.Write(DEFAULT_FILE_NAME)
	default:
		fmt.Println(GetHelpString())
	}
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

func diff() {
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
	pull()
}

type KeyTuple struct {
	Key         string
	Environment string
}

func pull() {
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

func import_gl() {
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

func push() {
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
