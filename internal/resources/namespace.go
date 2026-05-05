// Copyright Tilaa B.V. 2026
// SPDX-License-Identifier: MPL-2.0

package resources

import (
	"context"
	"time"

	"github.com/nexaa-cloud/nexaa-cli/api"
)

func waitForNamespaceToBeRemoved(ctx context.Context, client api.Client, namespaceName string) error {
	const (
		initialDelay = 2 * time.Second
		maxDelay     = 15 * time.Second
	)
	delay := initialDelay

	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		_, err := client.NamespaceListByName(namespaceName)
		if err != nil {
			// Namespace no longer found — deletion complete
			return nil
		}

		time.Sleep(delay)
		if delay < maxDelay {
			delay *= 2
			if delay > maxDelay {
				delay = maxDelay
			}
		}
	}
}

func waitForAllChildrenToBeRemoved(ctx context.Context, client api.Client, namespaceName string) error {
	const (
		initialDelay = 2 * time.Second
		maxDelay     = 15 * time.Second
	)
	delay := initialDelay

	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		namespace, err := client.NamespaceListByName(namespaceName)
		if err != nil {
			return err
		}

		hasQueues, err := namespaceHasMessageQueues(client, namespaceName)
		if err != nil {
			return err
		}

		if !namespaceHasContainers(namespace) && !namespaceHasContainerJobs(namespace) && !namespaceHasCloudDatabaseClusters(namespace) && !namespaceHasVolumes(namespace) && !hasQueues {
			break
		}

		// Backoff between polls
		time.Sleep(delay)
		if delay < maxDelay {
			delay *= 2
			if delay > maxDelay {
				delay = maxDelay
			}
		}
	}

	return nil
}

func namespaceHasContainers(namespace api.NamespaceResult) bool {
	return len(namespace.Containers) != 0
}

func namespaceHasContainerJobs(namespace api.NamespaceResult) bool {
	return len(namespace.ContainerJobs) != 0
}

func namespaceHasVolumes(namespace api.NamespaceResult) bool {
	return len(namespace.Volumes) != 0
}

func namespaceHasCloudDatabaseClusters(namespace api.NamespaceResult) bool {
	return len(namespace.CloudDatabaseClusters) != 0
}

func namespaceHasMessageQueues(client api.Client, namespaceName string) (bool, error) {
	queues, err := client.MessageQueueList()
	if err != nil {
		return false, err
	}
	for _, q := range queues {
		if q.GetNamespace().Name == namespaceName {
			return true, nil
		}
	}
	return false, nil
}
