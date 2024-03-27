package main

import (
	"testing"
)

// TestHelloName calls greetings.Hello with a name, checking
// for a valid return value.
func TestNesting(t *testing.T) {
	project := ProjectSecrets{
		ProjectID: 1,
		Variables: []Secret{
			{
				Key:          "key1",
				Value:        "value1",
				Description:  "description1",
				VariableType: "variableType1",
				Environment:  "environment1",
				Protect:      false,
				Mask:         false,
				Raw:          false,
			},
		},
	}
	nested := NestedProjectSecrets{
		ProjectID: 1,
		Variables: []NestedSecret{
			{
				Key:          "key1",
				Value:        &project.Variables[0].Value,
				Description:  &project.Variables[0].Description,
				VariableType: &project.Variables[0].VariableType,
				Environment:  &project.Variables[0].Environment,
				Protect:      &project.Variables[0].Protect,
				Mask:         &project.Variables[0].Mask,
				Raw:          &project.Variables[0].Raw,
			},
		},
	}

	result := project.Nest()
	if !(nested.ProjectID == result.ProjectID) {
		t.Fatalf(`project.Nest() = %v, want %v`, result, nested)
	}
	for i := 0; i < len(nested.Variables); i++ {
		if !(nested.Variables[i].Key == result.Variables[i].Key) {
			t.Fatalf(`project.Nest() key = %v, want %v`, result.Variables[i].Key, nested.Variables[i].Key)
		}
		if !(*nested.Variables[i].Value == *result.Variables[i].Value) {
			t.Fatalf(`project.Nest() value = %v, want %v`, *result.Variables[i].Value, *nested.Variables[i].Value)
		}
		if !(*nested.Variables[i].Description == *result.Variables[i].Description) {
			t.Fatalf(`project.Nest() description = %v, want %v`, *result.Variables[i].Description, *nested.Variables[i].Description)
		}
		if !(*nested.Variables[i].VariableType == *result.Variables[i].VariableType) {
			t.Fatalf(`project.Nest() vartype = %v, want %v`, *result.Variables[i].VariableType, *nested.Variables[i].VariableType)
		}
		if !(*nested.Variables[i].Environment == *result.Variables[i].Environment) {
			t.Fatalf(`project.Nest() env = %v, want %v`, *result.Variables[i].Environment, *nested.Variables[i].Environment)
		}
		if !(*nested.Variables[i].Protect == *result.Variables[i].Protect) {
			t.Fatalf(`project.Nest() protect = %v, want %v`, *result.Variables[i].Protect, *nested.Variables[i].Protect)
		}
		if !(*nested.Variables[i].Mask == *result.Variables[i].Mask) {
			t.Fatalf(`project.Nest() mask = %v, want %v`, *result.Variables[i].Mask, *nested.Variables[i].Mask)
		}
		if !(*nested.Variables[i].Raw == *result.Variables[i].Raw) {
			t.Fatalf(`project.Nest() raw = %v, want %v`, *result.Variables[i].Raw, *nested.Variables[i].Raw)
		}
	}
}
