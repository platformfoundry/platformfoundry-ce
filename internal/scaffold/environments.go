package scaffold

import (
	"path/filepath"
)

const devEnvironmentTemplate = `
name: dev
production: false
domain: dev.example.com
`

const stagingEnvironmentTemplate = `
name: staging
production: false
domain: staging.example.com
`

const prodEnvironmentTemplate = `
name: prod
production: true
domain: example.com
`

func (g *Generator) generateEnvironments(config ScaffoldConfig) (*GenerateResult, error) {
	result := &GenerateResult{
		Files: make([]GeneratedFile, 0),
	}

	environments := map[string]string{
		"dev.yaml":     devEnvironmentTemplate,
		"staging.yaml": stagingEnvironmentTemplate,
		"prod.yaml":    prodEnvironmentTemplate,
	}

	for filename, content := range environments {
		file := GeneratedFile{
			Path:    filepath.Join(config.OutputDir, "environments", filename),
			Content: content,
		}

		if !config.DryRun {
			if err := g.writeFile(file.Path, content, config.Overwrite); err != nil {
				return nil, err
			}
			file.Created = true
		}
		result.Files = append(result.Files, file)
	}

	return result, nil
}
