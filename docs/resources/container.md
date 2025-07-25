---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "nexaa_container Resource - nexaa"
subcategory: ""
description: |-
  Container resource representing a container that will be deployed on nexaa.
---

# nexaa_container (Resource)

Container resource representing a container that will be deployed on nexaa.

## Example Usage

```terraform
resource "nexaa_container" "container" {
  name      = "tf-container"
  namespace = "terraform"
  image     = "nginx:latest"
  registry  = "gitlab"

  resources = {
    cpu = 0.25
    ram = 0.5
  }

  ports = ["8000:8000", "80:80"]

  environment_variables = [
    {
      name   = "ENV"
      value  = "production"
      secret = false
    },
    {
      name   = "Variable"
      value  = "terraform"
      secret = false
    },
    {
      name   = "API_KEY"
      value  = "supersecret"
      secret = true
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

  mounts = [
    {
      path   = "/storage/mount"
      volume = "storage"
    }
  ]

  health_check = {
    port = 80
    path = "/storage/health"
  }

  scaling = {
    type = "auto"

    #manual_input = 1
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
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `image` (String) The image use to run the container
- `name` (String) Name of the container
- `namespace` (String) Name of the namespace that the container will belong to
- `resources` (Attributes) The resources used for running the container (see [below for nested schema](#nestedatt--resources))
- `scaling` (Attributes) Used to specify or automaticaly scale the amount of replicas running (see [below for nested schema](#nestedatt--scaling))

### Optional

- `environment_variables` (Attributes List) Environment variables used in the container, write the non-secrets first the the secrets (see [below for nested schema](#nestedatt--environment_variables))
- `health_check` (Attributes) (see [below for nested schema](#nestedatt--health_check))
- `ingresses` (Attributes List) Used to access the container from the internet (see [below for nested schema](#nestedatt--ingresses))
- `mounts` (Attributes List) Used to add persistent storage to your container (see [below for nested schema](#nestedatt--mounts))
- `ports` (List of String) The ports used to expose for traffic, format as from:to
- `registry` (String) The registry used to be able to acces images that are saved in a private environment, fill in null to use a public registry

### Read-Only

- `id` (String) Unique identifier of the container, equal to the name
- `last_updated` (String) Timestamp of the last Terraform update of the private registry
- `status` (String) The status of the container

<a id="nestedatt--resources"></a>
### Nested Schema for `resources`

Required:

- `cpu` (Number) The amount of cpu used for the container, can be the following values: 0.25, 0.5, 0.75, 1, 2, 3, 4
- `ram` (Number) The amount of ram used for the container (in GB), can be the following values: 0.5, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16


<a id="nestedatt--scaling"></a>
### Nested Schema for `scaling`

Required:

- `type` (String) The type of scaling you want, auto or manual

Optional:

- `auto_input` (Attributes) The input for the autoscaling (see [below for nested schema](#nestedatt--scaling--auto_input))
- `manual_input` (Number) The input for manual scaling, equal to the amount of running replicas you want

<a id="nestedatt--scaling--auto_input"></a>
### Nested Schema for `scaling.auto_input`

Required:

- `maximal_replicas` (Number) The maximum amount of replicas you want to scale to
- `minimal_replicas` (Number) The minimal amount of replicas you want

Optional:

- `triggers` (Attributes List) Used as condition as to when the container needs to add a replica, you can have 2 triggers, one for eacht type (see [below for nested schema](#nestedatt--scaling--auto_input--triggers))

<a id="nestedatt--scaling--auto_input--triggers"></a>
### Nested Schema for `scaling.auto_input.triggers`

Required:

- `threshold` (Number) The amount percentage wise needed to add another replica
- `type` (String) The type of metric used for specifying what the triggers monitors, is eihter MEMORY or CPU




<a id="nestedatt--environment_variables"></a>
### Nested Schema for `environment_variables`

Required:

- `name` (String) The name used for the environment variable

Optional:

- `secret` (Boolean) A boolean to represent if the environment variable is a secret or not
- `value` (String) The value used for the environment variable, is required


<a id="nestedatt--health_check"></a>
### Nested Schema for `health_check`

Required:

- `path` (String)
- `port` (Number)


<a id="nestedatt--ingresses"></a>
### Nested Schema for `ingresses`

Required:

- `port` (Number) The port used for the ingress, must be one of the exposed ports
- `tls` (Boolean) Boolean representing if you want TLS enabled or not

Optional:

- `allow_list` (List of String) A list with the IP's that can access the ingress url, 0.0.0.0/0 to make it accessible for everyone
- `domain_name` (String) The domain used for the ingress, defaults to https://101010-{namespaceName}-{containerName}.container.tilaa.cloud


<a id="nestedatt--mounts"></a>
### Nested Schema for `mounts`

Required:

- `path` (String) The path to the location where the data will be saved
- `volume` (String) The name of the volume that is used for the mount
