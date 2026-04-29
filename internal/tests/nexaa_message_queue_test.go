// Copyright Tilaa B.V. 2026
// SPDX-License-Identifier: MPL-2.0

package tests

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func messageQueueConfig(namespaceName string, queueName string, mqType string, version string, cpu string, memory string, storage string, replicas string, allowlist []string) string {
	return givenProvider() +
		givenNamespace(namespaceName, "") +
		givenMessageQueue(queueName, mqType, version, cpu, memory, storage, replicas, allowlist)
}

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
				Config: messageQueueConfig(namespaceName, queueName, "RabbitMQ", "3.13", "0.25", "0.5", "5.0", "1", []string{"192.168.1.1", "192.168.1.2"}),
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
					resource.TestCheckResourceAttr("nexaa_message_queue.queue", "allowlist.#", "2"),
					resource.TestCheckResourceAttr("nexaa_message_queue.queue", "allowlist.0", "127.0.0.1"),
					resource.TestCheckResourceAttr("nexaa_message_queue.queue", "allowlist.1", "192.168.1.1"),
					resource.TestCheckResourceAttrSet("nexaa_message_queue.queue", "external_connection.ipv4"),
					resource.TestCheckResourceAttrSet("nexaa_message_queue.queue", "external_connection.ipv6"),
					resource.TestCheckResourceAttr("nexaa_message_queue.queue", "external_connection.ports.allowlist.#", "2"),
					resource.TestCheckResourceAttrSet("nexaa_message_queue.queue", "external_connection.ports.external_port"),
				),
			},

			// 2) ImportState
			{
				ResourceName:            "nexaa_message_queue.queue",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateId:           fmt.Sprintf("%s/%s", namespaceName, queueName),
				ImportStateVerifyIgnore: []string{"last_updated", "plan", "type", "version", "allowlist"},
			},

			// 3) Update & Read
			{
				Config: messageQueueConfig(namespaceName, queueName, "RabbitMQ", "3.13", "0.25", "0.5", "5.0", "1", []string{"192.168.1.1"}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("nexaa_message_queue.queue", "name", queueName),
					resource.TestCheckResourceAttr("nexaa_message_queue.queue", "namespace", namespaceName),
					resource.TestCheckResourceAttr("nexaa_message_queue.queue", "external_connection.ports.allowlist.#", "1"),
				),
			},
			// 4) Delete is automatically tested by TestCase
		},
	})
}
