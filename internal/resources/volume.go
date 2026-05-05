// Copyright Tilaa B.V. 2026
// SPDX-License-Identifier: MPL-2.0

package resources

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nexaa-cloud/nexaa-cli/api"
)

func translateApiToVolumeResource(plan volumeResource, volume api.VolumeResult) volumeResource {
	plan.ID = types.StringValue(generateNamespaceVolumeId(plan.Namespace.ValueString(), volume.GetName()))
	plan.Namespace = types.StringValue(plan.Namespace.ValueString())
	plan.Name = types.StringValue(volume.Name)
	plan.Size = types.Int64Value(int64(volume.Size))
	plan.Usage = types.Float64Value(volume.Usage)
	plan.Locked = types.BoolValue(volume.Locked)
	plan.Status = types.StringValue(volume.State)
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))
	return plan
}

func waitForAllContainersToBeUnmounted(ctx context.Context, client api.Client, namespace string, volumeName string) error {
	const (
		initialDelay = 2 * time.Second
		maxDelay     = 15 * time.Second
	)
	delay := initialDelay

	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		volume, err := client.ListVolumeByName(namespace, volumeName)
		if err != nil {
			return err
		}

		if volume == nil {
			return fmt.Errorf("volume %s not found", volumeName)
		}

		if !hasContainersAttached(*volume) && !hasContainerJobsAttached(*volume) {
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

func hasContainersAttached(volume api.VolumeResult) bool {
	return len(volume.Containers) != 0
}

func hasContainerJobsAttached(volume api.VolumeResult) bool {
	return len(volume.ContainerJobs) != 0
}