// Copyright Tilaa B.V. 2026
// SPDX-License-Identifier: MPL-2.0

package resources

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
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

func volumeLocked() fetchResourceLocked {
	return func(client api.Client, namespace string, resourceName string) (bool, error) {
		resource, err := client.ListVolumeByName(
			namespace,
			resourceName,
		)

		if err != nil {
			return false, err
		}

		// ListVolumeByName returns a pointer; treat a nil result (volume gone)
		// as unlocked so callers proceed to the next step instead of panicking.
		if resource == nil {
			return false, nil
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

func messageQueueLocked() fetchResourceLocked {
	return func(client api.Client, namespace string, resourceName string) (bool, error) {
		resource, err := client.MessageQueueGet(
			api.MessageQueueResourceInput{
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

func registryLocked() fetchResourceLocked {
	return func(client api.Client, namespace string, resourceName string) (bool, error) {
		resource, err := client.ListRegistryByName(
			namespace,
			resourceName,
		)

		if err != nil {
			return false, err
		}

		return resource.Locked, nil
	}
}

func waitForUnlocked(ctx context.Context, fetchResourceLocked fetchResourceLocked, client api.Client, namespace string, resourceName string) error {
	const (
		initialDelay = 2 * time.Second
		maxDelay     = 15 * time.Second
	)
	delay := initialDelay

	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		// The Nexaa SDK ignores caller context, so a stuck request would
		// otherwise pin this loop forever. Run the poll in a goroutine and
		// race it against ctx.Done so cancellation always wins promptly.
		// A hung fetch still runs in the background and will be GC'd when
		// it eventually returns.
		type pollResult struct {
			locked bool
			err    error
		}
		ch := make(chan pollResult, 1)
		go func() {
			locked, err := fetchResourceLocked(client, namespace, resourceName)
			ch <- pollResult{locked: locked, err: err}
		}()

		select {
		case <-ctx.Done():
			return ctx.Err()
		case res := <-ch:
			if res.err != nil {
				return res.err
			}
			if !res.locked {
				return nil
			}
			tflog.Info(ctx, resourceName+" is locked, retrying")
		}

		// Cancellable backoff between polls.
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}

		if delay < maxDelay {
			delay *= 2
			if delay > maxDelay {
				delay = maxDelay
			}
		}
	}
}
