package extractors

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types"
	"github.com/headingy/trireme/constants"
	"github.com/headingy/trireme/policy"

	dockerClient "github.com/docker/docker/client"
)

// SwarmExtractor is an example metadata extractor for swarm that uses the service
// labels for policy decisions
func SwarmExtractor(info *types.ContainerJSON) (*policy.PURuntime, error) {

	// Create a docker client
	defaultHeaders := map[string]string{"User-Agent": "engine-api-dockerClient-1.0"}
	cli, err := dockerClient.NewClient("unix:///var/run/docker.sock", "v1.23", nil, defaultHeaders)
	if err != nil {
		return nil, fmt.Errorf("Error creating Docker client %s", err)
	}

	// Get the labels from Docker. If it is a swarm service, get the labels from
	// the service definition instead.
	dockerLabels := info.Config.Labels
	if _, ok := info.Config.Labels["com.docker.swarm.service.id"]; ok {

		serviceID := info.Config.Labels["com.docker.swarm.service.id"]

		service, _, err := cli.ServiceInspectWithRaw(context.Background(), serviceID)
		if err != nil {
			return nil, fmt.Errorf("Failed get swarm labels: %s", err)
		}

		dockerLabels = service.Spec.Labels
	}

	// Create the tags based on the docker labels
	tags := policy.NewTagsMap(map[string]string{
		"image": info.Config.Image,
		"name":  info.Name,
	})
	for k, v := range dockerLabels {
		tags.Add(k, v)
	}

	ipa := policy.NewIPMap(map[string]string{
		"bridge": "0.0.0.0/0",
	})

	return policy.NewPURuntime(info.Name, info.State.Pid, tags, ipa, constants.ContainerPU, nil), nil
}
