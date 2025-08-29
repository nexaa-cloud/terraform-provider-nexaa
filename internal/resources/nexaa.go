package resources

import (
	"errors"
	"fmt"
	"strings"
)

type namespaceChildId struct {
	Namespace string
	Name      string
}

func generateNamespaceVolumeId(namespace string, name string) string {
	return fmt.Sprintf("%s/volume/%s", namespace, name)
}

func unpackNamespaceChildId(id string) (namespaceChildId, error) {
	parts := strings.SplitN(id, "/", 3)
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		return namespaceChildId{}, errors.New(
			"Expected import ID in the format \"<namespace>/<type_name>/<child_name>\", got: " + id,
		)
	}

	namespace := parts[0]
	childName := parts[2]

	return namespaceChildId{
		Namespace: namespace,
		Name:      childName,
	}, nil
}
