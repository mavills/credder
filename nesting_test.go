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
			{
				Key:          "key1",
				Value:        "value2",
				Description:  "description1",
				VariableType: "variableType1",
				Environment:  "environment2",
				Protect:      true,
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
				Description:  &project.Variables[0].Description,
				VariableType: &project.Variables[0].VariableType,
				Mask:         &project.Variables[0].Mask,
				Raw:          &project.Variables[0].Raw,
				Nested: []NestedSecret{
					{
						Value:       &project.Variables[0].Value,
						Environment: &project.Variables[0].Environment,
						Protect:     &project.Variables[0].Protect,
					},
					{
						Value:       &project.Variables[1].Value,
						Environment: &project.Variables[1].Environment,
						Protect:     &project.Variables[1].Protect,
					},
				},
			},
		},
	}

	// Check that the nested project is equal
	result := project.Nest()
	if !(nested.Equal(result)) {
		t.Fatalf(`project.Nest() = %+v, want %+v`, result, nested)
	}
	// Check that unnesting is also ok
	unnested := result.Unnest()
	if !project.Equal(unnested) {
		t.Fatalf(`result.Unnest() = %v, want %v`, unnested, project)
	}
}
