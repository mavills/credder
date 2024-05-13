package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/go-yaml/yaml"
)

// Linting consists of 2 passes:
// 1. Merge all includes and send to gitlab lint api
// 2. Apply extra linting rules
//    - check variables
//    - check args of helm install

// When linting all network requests are cached in /tmp
// To clear the cache, run `credder lint clear-cache`

// Some things to consider while doing the first pass
// - you can override a job with the same name if the job comes from an external include

var lintCache map[string]string = make(map[string]string)

func GetLintCache(key string) (string, bool) {
	lookup, ok := lintCache[key]
	return lookup, ok
}

func GetLintCacheB(key string) ([]byte, bool) {
	lookup, ok := lintCache[key]
	return []byte(lookup), ok
}

func GetLintCacheI(key string) (int, bool) {
	lookup, ok := lintCache[key]

	if !ok {
		return 0, false
	}

	value, err := strconv.Atoi(lookup)
	if err != nil {
		return 0, false
	}
	return value, true
}

func SetLintCache(key string, value any) {
	stringValue, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	lintCache[key] = string(stringValue)
}

func SetLintCacheS(key string, value string) {
	lintCache[key] = value
}

func SetLintCacheI(key string, value int) {
	stringValue := strconv.Itoa(value)
	lintCache[key] = stringValue
}

type IncludeEntry struct {
	Project string `yaml:"project"`
	Local   string `yaml:"local"`
	File    string `yaml:"file"`
}

func (i *IncludeEntry) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var str string
	if err := unmarshal(&str); err == nil {
		i.Local = str
		return nil
	}
	type rawIncludeEntry IncludeEntry
	return unmarshal((*rawIncludeEntry)(i))
}

type IncludeStageFile struct {
	Include []IncludeEntry `yaml:"include"`
}

func loadCache() error {
	bytes, err := os.ReadFile("/tmp/credder-lint-cache")
	if err == nil {
		err := json.Unmarshal(bytes, &lintCache)
		if err != nil {
			return fmt.Errorf("error unmarshalling json: %w", err)
		}

	}
	return nil
}

func getContentFromIncludes(yamlString string) (string, error) {
	var t IncludeStageFile
	err := yaml.Unmarshal([]byte(yamlString), &t)
	if err != nil {
		return "", fmt.Errorf("error unmarshalling yaml: %w", err)
	}
	allContent := yamlString
	for _, include := range t.Include {
		if include.Project != "" {
			project_id, err := GetProjectIdFromPath(include.Project)
			if err != nil {
				return "", fmt.Errorf("error getting project id from path: %w", err)
			}
			content, err := GetFileFromProjectIdAndPath(project_id, include.File)
			if err != nil {
				return "", fmt.Errorf("error getting file from project id and path: %w", err)
			}
			includeContent, err := getContentFromIncludes(content)
			if err != nil {
				return "", fmt.Errorf("error getting content from includes: %w", err)
			}
			allContent = allContent + "\n" + includeContent
		} else if include.Local != "" {
			wd, _ := os.Getwd()
			content, err := os.ReadFile(wd + include.Local)
			if err != nil {
				return "", fmt.Errorf("error reading file: %w", err)
			}
			includeContent, err := getContentFromIncludes(string(content))
			if err != nil {
				return "", fmt.Errorf("error getting content from includes: %w", err)
			}

			allContent = allContent + "\n" + includeContent

		}
	}
	return allContent, nil
}

// recusively get all includes
// lint
// repeat for possible trigger pipelines
func pass1(path string, configuration string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("error reading file: %w", err)
	}

	content, err := getContentFromIncludes(string(data))
	if err != nil {
		return fmt.Errorf("error getting content from includes: %w", err)
	}

	lintResult, err := LintCiFromString(content)
	if err != nil {
		return fmt.Errorf("error linting ci from string: %w", err)
	}

	fmt.Printf("=============== %s =================\n", configuration)
	if lintResult.Valid {
		fmt.Println("Valid :)")
	} else {
		fmt.Println("Errors: ")
		for _, e := range lintResult.Errors {
			fmt.Println("=>", e)
		}
	}

	// Get trigger pipelines
	return nil
}

func Lint() error {
	err := loadCache()
	if err != nil {
		return fmt.Errorf("error loading cache: %w", err)
	}

	projectId := GetProjectID()
	ciConfigPath, err := GetCiConfigPath(projectId)
	if err != nil {
		return fmt.Errorf("error getting CI config path: %w", err)
	}

	err = pass1(ciConfigPath, "Main configuration")
	if err != nil {
		return fmt.Errorf("error in pass 1: %w", err)
	}

	// Write lint cache to /tmp/credder-lint-cache
	var bytes []byte
	bytes, _ = json.Marshal(lintCache)
	os.WriteFile("/tmp/credder-lint-cache", bytes, 0644)

	return nil
}
