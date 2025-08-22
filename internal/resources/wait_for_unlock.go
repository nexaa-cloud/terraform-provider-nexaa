package resources

import (
	"context"
	"time"

	"github.com/nexaa-cloud/nexaa-cli/api"
)

type fetchResourceLocked func(client api.Client, namespace string, resourceName string) (bool, error)

func containerLocked() fetchResourceLocked {
	return func(client api.Client, namespace string, resourceName string) (bool, error) {
		resource, err := client.ListContainerByName(
			namespace,
			resourceName,
		)

		if err != nil {
			return false, err
		}

		return resource.Locked, nil
	}
}

func containerJobLocked() fetchResourceLocked {
	return func(client api.Client, namespace string, resourceName string) (bool, error) {
		resource, err := client.ContainerJobByName(
			namespace,
			resourceName,
		)

		if err != nil {
			return false, err
		}

		return resource.Locked, nil
	}
}

func cloudDatabaseClusterLocked() fetchResourceLocked {
	return func(client api.Client, namespace string, resourceName string) (bool, error) {
		resource, err := client.CloudDatabaseClusterGet(
			api.CloudDatabaseClusterResourceInput{
				Namespace: namespace,
				Name:      resourceName,
			},
		)

		if err != nil {
			return false, err
		}

		return resource.Locked, nil
	}
}

func waitForUnlocked(ctx context.Context, fetchResourceLocked fetchResourceLocked, client api.Client, namespace string, resourceName string) (bool, error) {
	const (
		initialDelay = 2 * time.Second
		maxDelay     = 15 * time.Second
	)
	delay := initialDelay

	for {
		if ctx.Err() != nil {
			return false, ctx.Err()
		}

		locked, err := fetchResourceLocked(client, namespace, resourceName)

		if err != nil {
			return false, err
		}

		if locked == false {
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

	return true, nil
}
