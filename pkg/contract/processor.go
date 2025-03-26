package contract

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"

	"buildbear-contract-verification-tool/pkg/utils"
)

// FindArtifactPath finds the artifact path for a given contract name
func FindArtifactPath(outDir, contractName string) (string, error) {
	var artifactPath string
	err := filepath.Walk(outDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && info.Name() == contractName+".json" {
			artifactPath = path
			return filepath.SkipDir
		}
		return nil
	})

	if err != nil {
		return "", err
	}

	if artifactPath == "" {
		return "", utils.LogErrorf("artifact not found for contract %s", contractName)
	}

	return artifactPath, nil
}

// ProcessSources processes the sources from the metadata
func ProcessSources(sources map[string]interface{}) (string, error) {
	if sources == nil {
		log.Println("No sources provided")
		return "{}", nil
	}

	transformedSources := make(map[string]map[string]string)

	for filePath, sourceInfo := range sources {
		sourceMap, ok := sourceInfo.(map[string]interface{})
		if !ok {
			continue
		}

		// Check if content is already in the metadata
		if content, ok := sourceMap["content"].(string); ok {
			transformedSources[filePath] = map[string]string{"content": content}
			continue
		}

		// If no content in metadata, try to read the file directly
		absolutePath, err := filepath.Abs(filePath)
		if err != nil {
			log.Printf("Error resolving absolute path for %s: %v\n", filePath, err)
			continue
		}

		content, err := os.ReadFile(absolutePath)
		if err != nil {
			log.Printf("Error reading file %s: %v\n", filePath, err)

			// Try alternative path - sometimes lib paths need to be resolved differently
			if len(filePath) > 4 && filePath[:4] == "lib/" {
				nodeModulesPath := filepath.Join("node_modules", filePath[4:])
				content, err = os.ReadFile(nodeModulesPath)
				if err != nil {
					// Fall back to a placeholder
					transformedSources[filePath] = map[string]string{"content": utils.LogErrorf("// Content for %s not available", filePath).Error()}
				} else {
					transformedSources[filePath] = map[string]string{"content": string(content)}
				}
			} else {
				// Fall back to a placeholder
				transformedSources[filePath] = map[string]string{"content": utils.LogErrorf("// Content for %s not available", filePath).Error()}
			}
		} else {
			transformedSources[filePath] = map[string]string{"content": string(content)}
		}
	}

	result, err := json.MarshalIndent(transformedSources, "", "  ")
	if err != nil {
		return "{}", err
	}

	return string(result), nil
}

// ProcessRemappings processes the remappings from the metadata
func ProcessRemappings(remappings interface{}) (string, error) {
	if remappings == nil {
		return "[]", nil
	}

	remappingsSlice, ok := remappings.([]interface{})
	if !ok {
		return "[]", nil
	}

	result, err := json.MarshalIndent(remappingsSlice, "", "  ")
	if err != nil {
		return "[]", err
	}

	return string(result), nil
}
