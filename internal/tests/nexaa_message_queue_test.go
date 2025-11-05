// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tests

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAcc_MessageQueueResource_basic(t *testing.T) {
	testAccPreCheck(t)

	// Generate random test data
	namespaceName := generateTestNamespace()
	queueName := generateTestMessageQueueName()

	t.Logf("=== MESSAGE QUEUE TEST USING NAMESPACE: %s ===", namespaceName)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// 1) Create & Read
			{
				Config: givenProvider() +
					givenNamespace(namespaceName, "") +
					givenMessageQueue(queueName, "RabbitMQ", "3.13", "1", "2", "10", "1"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("nexaa_message_queue.queue", "id"),
					resource.TestCheckResourceAttr("nexaa_message_queue.queue", "namespace", namespaceName),
					resource.TestCheckResourceAttr("nexaa_message_queue.queue", "name", queueName),
					resource.TestCheckResourceAttr("nexaa_message_queue.queue", "type", "RabbitMQ"),
					resource.TestCheckResourceAttr("nexaa_message_queue.queue", "version", "3.13"),
					resource.TestCheckResourceAttrSet("nexaa_message_queue.queue", "plan"),
					resource.TestCheckResourceAttrSet("nexaa_message_queue.queue", "state"),
					resource.TestCheckResourceAttrSet("nexaa_message_queue.queue", "locked"),
					resource.TestCheckResourceAttrSet("nexaa_message_queue.queue", "last_updated"),
				),
			},

			// 2) ImportState
			{
				ResourceName:            "nexaa_message_queue.queue",
				ImportState:             true,
				ImportStateId:           fmt.Sprintf("%s/%s", namespaceName, queueName),
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"last_updated", "plan", "type", "version"},
			},
		},
	})
}
