// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tests

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func containerConfig() string {
	return providerConfig + `
resource "nexaa_namespace" "ns" {
  name = "tf-test-con"
}

resource "nexaa_registry" "registry" {
  namespace = "tf-test-con"
  name      = "example"
  source    = "registry.gitlab.com"
  username  = "user"
  password  = "pass"
  verify    = false
}

resource "nexaa_volume" "volume1" {
  namespace      = "tf-test-con"
  name           = "tf-vol"
  size           = 2
  }
  

resource "nexaa_container" "container" {
  name      = "tf-container"
  namespace = nexaa_namespace.ns.name
  image     = "nginx:latest"
  registry  = null

  resources = {
    cpu = 0.25
    ram = 0.5
  }

  ports = ["80:80"]

  environment_variables = [
    {
      name   = "Variable"
      value  = "terraform"
      secret = false
    }
  ]

  ingresses = [
    {
      domain_name = null
      port        = 80
      tls         = true
      allow_list  = ["0.0.0.0/0"]
    }
  ]

  health_check = {
    port = 80
    path = "/storage/health"
  }

  scaling = {
    type = "auto"
    auto_input = {
      minimal_replicas = 1
      maximal_replicas = 3

      triggers = [
        {
          type      = "CPU"
          threshold = 70
        },
        {
          type      = "MEMORY"
          threshold = 80
        }
      ]
    }
  }
}
`
}

func containerUpdateConfig() string {
	return providerConfig + `
resource "nexaa_namespace" "ns" {
  name = "tf-test-con"
}

resource "nexaa_volume" "volume1" {
  namespace      = "tf-test-con"
  name           = "tf-vol"
  size           = 2
}

resource "nexaa_registry" "registry" {
  namespace = "tf-test-con"
  name      = "example"
  source    = "registry.gitlab.com"
  username  = "user"
  password  = "pass"
  verify    = false
}

resource "nexaa_container" "container" {
  name      = "tf-container"
  namespace = nexaa_namespace.ns.name
  image     = "nginx:alpine"
  registry  = "example"

  resources = {
    cpu = 0.5
    ram = 1.0
  }

  ports = ["80:80", "8000:8000"]

  environment_variables = [
    {
      name   = "Variable"
      value  = "terraform"
      secret = false
    },
    {
      name   = "ENV"
      value  = "staging"
      secret = false
    }
  ]

  ingresses = [
    {
      domain_name = null
      port        = 80
      tls         = true
      allow_list  = ["0.0.0.0/0"]
    }
  ]

  health_check = {
    port = 80
    path = "/health"
  }

  scaling = {
    type = "manual"
    manual_input = 3
  }
}
`
}

func TestAcc_ContainerResource_basic(t *testing.T) {
	if os.Getenv("NEXAA_USERNAME") == "" || os.Getenv("NEXAA_PASSWORD") == "" {
		t.Fatal("NEXAA_USERNAME and NEXAA_PASSWORD must be set")
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// 1) Create
			{
				Config: containerConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("nexaa_container.container", "id"),
					resource.TestCheckResourceAttr("nexaa_container.container", "name", "tf-container"),
					resource.TestCheckResourceAttr("nexaa_container.container", "namespace", "tf-test-con"),
					resource.TestCheckResourceAttr("nexaa_container.container", "image", "nginx:latest"),
					//resource.TestCheckResourceAttr("nexaa_container.container", "registry", "public"),
					resource.TestCheckResourceAttr("nexaa_container.container", "resources.cpu", "0.25"),
					resource.TestCheckResourceAttr("nexaa_container.container", "resources.ram", "0.5"),
					resource.TestCheckResourceAttr("nexaa_container.container", "ports.#", "1"),
					resource.TestCheckResourceAttr("nexaa_container.container", "environment_variables.#", "1"),
					resource.TestCheckResourceAttr("nexaa_container.container", "ingresses.#", "1"),
					//resource.TestCheckResourceAttr("nexaa_container.container", "mounts.#", "1"),
					resource.TestCheckResourceAttr("nexaa_container.container", "health_check.port", "80"),
					resource.TestCheckResourceAttr("nexaa_container.container", "health_check.path", "/storage/health"),
					resource.TestCheckResourceAttr("nexaa_container.container", "scaling.type", "auto"),
					resource.TestCheckResourceAttr("nexaa_container.container", "scaling.auto_input.minimal_replicas", "1"),
					resource.TestCheckResourceAttr("nexaa_container.container", "scaling.auto_input.maximal_replicas", "3"),
					resource.TestCheckResourceAttr("nexaa_container.container", "scaling.auto_input.triggers.#", "2"),
				),
			},

			// 2) ImportState
			{
				ResourceName:      "nexaa_container.container",
				ImportState:       true,
				ImportStateId:     "tf-test-con/tf-container",
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"registry",
					"mounts",
					"environment_variables.0.value",
					"environment_variables.1.value",
					"ingresses.0.domain_name",
					"last_updated",
				},
			},

			// 3) Update
			{
				Config: containerUpdateConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("nexaa_container.container", "image", "nginx:alpine"),
					resource.TestCheckResourceAttr("nexaa_container.container", "registry", "example"),
					resource.TestCheckResourceAttr("nexaa_container.container", "resources.cpu", "0.5"),
					resource.TestCheckResourceAttr("nexaa_container.container", "resources.ram", "1"),
					resource.TestCheckResourceAttr("nexaa_container.container", "ports.#", "2"),
					resource.TestCheckResourceAttr("nexaa_container.container", "environment_variables.#", "2"),
					resource.TestCheckResourceAttr("nexaa_container.container", "environment_variables.0.name", "Variable"),
					resource.TestCheckResourceAttr("nexaa_container.container", "environment_variables.0.value", "terraform"),
					resource.TestCheckResourceAttr("nexaa_container.container", "environment_variables.1.name", "ENV"),
					resource.TestCheckResourceAttr("nexaa_container.container", "environment_variables.1.value", "staging"),
					resource.TestCheckResourceAttr("nexaa_container.container", "ingresses.#", "1"),
					resource.TestCheckResourceAttr("nexaa_container.container", "ingresses.0.port", "80"),
					resource.TestCheckResourceAttr("nexaa_container.container", "ingresses.0.tls", "true"),
					resource.TestCheckResourceAttr("nexaa_container.container", "ingresses.0.allow_list.0", "0.0.0.0/0"),
					resource.TestCheckResourceAttr("nexaa_container.container", "mounts.#", "0"),
					resource.TestCheckResourceAttr("nexaa_container.container", "health_check.port", "80"),
					resource.TestCheckResourceAttr("nexaa_container.container", "health_check.path", "/health"),
					resource.TestCheckResourceAttr("nexaa_container.container", "scaling.type", "manual"),
					resource.TestCheckResourceAttr("nexaa_container.container", "scaling.manual_input", "3"),
				),
			},
		},
	})
}
