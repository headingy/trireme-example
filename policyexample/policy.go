package policyexample

import (
	"fmt"

	"github.com/headingy/trireme"
	"github.com/headingy/trireme/monitor"
	"github.com/headingy/trireme/policy"
	"go.uber.org/zap"
)

// CustomPolicyResolver is a simple policy engine
type CustomPolicyResolver struct {
	triremeNets []string
}

// NewCustomPolicyResolver creates a new example policy engine for the Trireme package
func NewCustomPolicyResolver(networks []string) *CustomPolicyResolver {

	return &CustomPolicyResolver{triremeNets: networks}
}

// ResolvePolicy implements the Trireme interface. Here we just create a simple
// policy that accepts packets with the same labels as the target container.
// We also add some egress/ingress services
func (p *CustomPolicyResolver) ResolvePolicy(context string, runtimeInfo policy.RuntimeReader) (*policy.PUPolicy, error) {

	zap.L().Info("Getting Policy for ContainerID",
		zap.String("containerID", context),
		zap.String("name", runtimeInfo.Name()),
	)

	tagSelectors := p.createRules(runtimeInfo)

	// Allow https access to github, but drop http access
	ingress := policy.NewIPRuleList([]policy.IPRule{

		policy.IPRule{
			Address:  "192.30.253.0/24",
			Port:     "80",
			Protocol: "TCP",
			Action:   policy.Reject,
		},

		policy.IPRule{
			Address:  "192.30.253.0/24",
			Port:     "443",
			Protocol: "TCP",
			Action:   policy.Accept,
		},
		policy.IPRule{
			Address:  "0.0.0.0/0",
			Port:     "",
			Protocol: "icmp",
			Action:   policy.Accept,
		},
		policy.IPRule{
			Address:  "0.0.0.0/0",
			Port:     "53",
			Protocol: "udp",
			Action:   policy.Accept,
		},
	})

	// Allow access to container from localhost
	egress := policy.NewIPRuleList([]policy.IPRule{
		policy.IPRule{
			Address:  "172.17.0.1/32",
			Port:     "80",
			Protocol: "TCP",
			Action:   policy.Accept,
		},
		policy.IPRule{
			Address:  "0.0.0.0/0",
			Port:     "",
			Protocol: "icmp",
			Action:   policy.Accept,
		},
	})

	// Use the bridge IP from Docker.
	ipl := policy.NewIPMap(map[string]string{})
	if ip, ok := runtimeInfo.DefaultIPAddress(); ok {
		ipl.IPs[policy.DefaultNamespace] = ip
	}

	identity := runtimeInfo.Tags()

	annotations := runtimeInfo.Tags()

	excluded := []string{}

	containerPolicyInfo := policy.NewPUPolicy(context, policy.Police, ingress, egress, nil, tagSelectors, identity, annotations, ipl, p.triremeNets, excluded, nil)

	return containerPolicyInfo, nil
}

// HandlePUEvent implements the corresponding interface. We have no
// state in this example
func (p *CustomPolicyResolver) HandlePUEvent(context string, eventType monitor.Event) {

	zap.L().Info("Handling container event",
		zap.String("containerID", context),
		zap.String("event", string(eventType)),
	)
}

// SetPolicyUpdater is used in order to register a pointer to the policyUpdater
// We don't implement policy updates in this example
func (p *CustomPolicyResolver) SetPolicyUpdater(pu trireme.PolicyUpdater) error {
	return nil
}

// CreateRuleDB creates a simple Rule DB that accepts packets from
// containers with the same labels as the instantiated container.
// If any of the labels matches, the packet is accepted.
func (p *CustomPolicyResolver) createRules(runtimeInfo policy.RuntimeReader) *policy.TagSelectorList {

	selectorList := &policy.TagSelectorList{
		TagSelectors: []policy.TagSelector{},
	}

	tags := runtimeInfo.Tags()
	for key, value := range tags.Tags {
		kv := policy.KeyValueOperator{
			Key:      key,
			Value:    []string{value},
			Operator: policy.Equal,
		}

		tagSelector := policy.NewTagSelector([]policy.KeyValueOperator{kv}, policy.Accept)
		selectorList.TagSelectors = append(selectorList.TagSelectors, *tagSelector)

	}

	// Add a default deny policy that rejects always from "namespace=bad"
	kv := policy.KeyValueOperator{
		Key:      "namespace",
		Value:    []string{"bad"},
		Operator: policy.Equal,
	}

	tagSelector := policy.NewTagSelector([]policy.KeyValueOperator{kv}, policy.Reject)
	selectorList.TagSelectors = append(selectorList.TagSelectors, *tagSelector)

	for i, selector := range selectorList.TagSelectors {
		for _, clause := range selector.Clause {
			zap.L().Info("Trireme policy for container",
				zap.String("name", runtimeInfo.Name()),
				zap.Int("c", i),
				zap.String("selector", fmt.Sprintf("%#v", clause)),
			)
		}
	}
	return selectorList

}
