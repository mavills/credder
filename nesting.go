package main

func (project *NestedProjectSecrets) Unnest() ProjectSecrets {
	unnestedProject := ProjectSecrets{
		ProjectID: project.ProjectID,
		Variables: []Secret{},
	}

	for _, parent := range project.Variables {
		if len(parent.Nested) == 0 {
			unnestedProject.Variables = append(unnestedProject.Variables, Secret{
				Key:          parent.Key,
				Value:        *parent.Value,
				Description:  *parent.Description,
				VariableType: *parent.VariableType,
				Environment:  *parent.Environment,
				Protect:      *parent.Protect,
				Mask:         *parent.Mask,
				Raw:          *parent.Raw,
			})
		} else {
			for _, nested := range parent.Nested {
				if parent.Value != nil {
					nested.Value = parent.Value
				}
				if parent.Description != nil {
					nested.Description = parent.Description
				}
				if parent.VariableType != nil {
					nested.VariableType = parent.VariableType
				}
				if parent.Environment != nil {
					nested.Environment = parent.Environment
				}
				if parent.Protect != nil {
					nested.Protect = parent.Protect
				}
				if parent.Mask != nil {
					nested.Mask = parent.Mask
				}
				if parent.Raw != nil {
					nested.Raw = parent.Raw
				}

				unnestedProject.Variables = append(unnestedProject.Variables, Secret{
					Key:          parent.Key,
					Value:        *nested.Value,
					Description:  *nested.Description,
					VariableType: *nested.VariableType,
					Environment:  *nested.Environment,
					Protect:      *nested.Protect,
					Mask:         *nested.Mask,
					Raw:          *nested.Raw,
				})
			}
		}
	}
	unnestedProject.Order()
	return unnestedProject
}

func (project *ProjectSecrets) Nest() NestedProjectSecrets {
	// Group secrets by key
	keymap := make(map[string][]NestedSecret)
	for _, secret := range project.Variables {
		nestedSecret := NestedSecret{
			Key:          secret.Key,
			Value:        &secret.Value,
			Description:  &secret.Description,
			VariableType: &secret.VariableType,
			Environment:  &secret.Environment,
			Protect:      &secret.Protect,
			Mask:         &secret.Mask,
			Raw:          &secret.Raw,
			Nested:       []NestedSecret{},
		}
		keymap[secret.Key] = append(keymap[secret.Key], nestedSecret)
	}

	// for each group, nest secrets
	topLevelGroup := []NestedSecret{}
	for key, secrets := range keymap {
		if len(secrets) == 1 {
			// no need to nest
			topLevelGroup = append(topLevelGroup, secrets[0])
			continue
		}
		parentSecret := NestedSecret{
			Key:          key,
			Value:        secrets[0].Value,
			Description:  secrets[0].Description,
			VariableType: secrets[0].VariableType,
			Environment:  secrets[0].Environment,
			Protect:      secrets[0].Protect,
			Mask:         secrets[0].Mask,
			Raw:          secrets[0].Raw,
			Nested:       []NestedSecret{},
		}
		// check for all fields if they are the same
		value := true
		description := true
		vartype := true
		env := true
		protect := true
		mask := true
		raw := true
		for _, secret := range secrets {
			if *secret.Value != *secrets[0].Value {
				parentSecret.Value = nil
				value = false
			}
			if *secret.Description != *secrets[0].Description {
				parentSecret.Description = nil
				description = false
			}
			if *secret.VariableType != *secrets[0].VariableType {
				parentSecret.VariableType = nil
				vartype = false
			}
			if *secret.Environment != *secrets[0].Environment {
				parentSecret.Environment = nil
				env = false
			}
			if *secret.Protect != *secrets[0].Protect {
				parentSecret.Protect = nil
				protect = false
			}
			if *secret.Mask != *secrets[0].Mask {
				parentSecret.Mask = nil
				mask = false
			}
			if *secret.Raw != *secrets[0].Raw {
				parentSecret.Raw = nil
				raw = false
			}
		}
		// Remove fields if they are the same
		for _, secret := range secrets {
			secret.Key = ""
			if value {
				secret.Value = nil
			}
			if description {
				secret.Description = nil
			}
			if vartype {
				secret.VariableType = nil
			}
			if env {
				secret.Environment = nil
			}
			if protect {
				secret.Protect = nil
			}
			if mask {
				secret.Mask = nil
			}
			if raw {
				secret.Raw = nil
			}
			parentSecret.Nested = append(parentSecret.Nested, secret)
		}

		topLevelGroup = append(topLevelGroup, parentSecret)
	}
	nestedProject := NestedProjectSecrets{
		ProjectID: project.ProjectID,
		Variables: topLevelGroup,
	}
	return nestedProject
}
