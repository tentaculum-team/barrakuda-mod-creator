package execute_init

import (
	"barrakudaModKit/internal/manifest"
)

// CreateAgent scaffolds an agent-type mod: a manifest.json declaring a
// docker block (Dockerfile + ports/resources/env) plus a starter Dockerfile.
// barrakuda-software builds and runs this image per the docker block —
// nothing else about the mod needs app code changes.
func CreateAgent(name string) (string, error) {
	m := manifest.Manifest{
		Name:        name,
		Version:     "0.0.1",
		Description: name + " agent mod",
		Icon:        "./img/icon.png",
		Banners:     []string{"./img/banner-1.png"},
		License:     "MIT",
		Type:        manifest.TypeAgent,
		Tags:        []string{"barrakuda", "agent"},
		Skill:       "./README.md",
		Docker: &manifest.DockerConfig{
			Dockerfile: "./docker/Dockerfile",
			Ports:      []manifest.DockerPort{{Host: 8080, Container: 8080}},
			Resources:  manifest.DockerResources{CPU: 1, Memory: "512m"},
			Env:        map[string]string{"EXAMPLE_VAR": "$user-name"},
		},
	}

	manifestJSON, err := m.JSON()
	if err != nil {
		return "", err
	}

	return writeFiles(name, map[string]string{
		"manifest.json":     manifestJSON,
		"docker/Dockerfile": agentDockerfile,
		"README.md":         "# " + name + "\n\nAgent mod: packaged as a Docker image, built from docker/Dockerfile and run by barrakuda-software per the `docker` block in manifest.json.\n",
	})
}

const agentDockerfile = `FROM alpine:3.20

# Replace with the real entrypoint for this agent.
CMD ["sh", "-c", "echo agent running && sleep infinity"]
`
