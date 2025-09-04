---
page_title: "Deploy your first container"
description: "How to deploy your first container on Nexaa"
---

# Deploy your first container
Let's walk through together on your first deploying !


## Prerequisites
***
- **Billing ready account**<br>
You have your account set up with billing enabled.<br> Don't have it yet ?
You can achieve that here: [Set up guide](https://docs.nexaa.io/getting-started/?utm_source=terraform)
- **Terraform/Open tofu installed**
- **Basic understanding of Terraform/Open Tofu commands**
- **Provider installed**
You have the provider installed. <br> Don't have it yet ?
You can find it here: [Setup provider](https://docs.nexaa.io/automation/terraform/?utm_source=terraform)

***

## Namespace
Before we start you will need a namespace, this is where your container will be living.
```terraform
resource "nexaa_namespace" "namespace" {
  name        = "namespace-name"
  description = "" # Description is optional
}
```

## Container
Now we can deploy a container, a container is an application which runs in an encapsulated environment. You can specify the amount of resources needed for your container.
This is a data source called `nexaa_container_resources`, where you can specify your cpu and memory. By default, the container has no persistent storage,
so all the data which has been saved in your container will be gone after an update.
```terraform

data "nexaa_container_resources" "container_resource" {
  cpu = 0.25
  memory = 0.5
}

resource "nexaa_container" "container" {
  depends_on = [
    nexaa_namespace.namespace
  ]
  name      = "container"
  namespace = nexaa_namespace.namespace.name
  image     = "nginx:latest"
  
  resources = data.nexaa_container_resources.container_resource.id
  
  ports = ["80:80"]

  ingresses = [
    {
      domain_name = "example.com" # domain_nane is optional if left empty we will provide one
      port        = 80
      tls         = true
      allow_list  = ["0.0.0.0/0"]
    }
  ]
}
```

### Persistent storage
Like said before a container doesn't have persistent storage. If you want to store data, we can add a volume to a container on a specific path
But first we need to deploy a volume.
```terraform
resource "nexaa_volume" "volume" {
  depends_on = [
    nexaa_namespace.namespace,
  ]
  namespace = nexaa_namespace.namespace
  name      = "volume"
  size      = 1 # the size of the volume can be 1 - 100, keep in mind the size can only grow not shrink.
}
```

After we have deployed our first volume we can mount it on a specific location in the container.
```terraform

data "nexaa_container_resources" "container_resource" {
  cpu = 0.25
  memory = 0.5
}

resource "nexaa_container" "container" {
  depends_on = [
    nexaa_namespace.namespace,
    nexaa_volume.volume
  ]
  name      = "container"
  namespace = nexaa_namespace.namespace.name
  image     = "nginx:latest"
  
  resources = data.nexaa_container_resources.container_resource.id
  
  ports = ["80:80"]

  ingresses = [
    {
      domain_name = "example.com" # domain_nane is optional if left empty we will provide one
      port        = 80
      tls         = true
      allow_list  = ["0.0.0.0/0"]
    }
  ]

  mounts = [
    {
      path   = "/var/wwww/html"
      volume = nexaa_volume.volume.name
    }
  ]
}
```

## Next Steps
- Explore our [marketplace](../marketplace.md)
- Explore the [complete documentation](https://docs.nexaa.io/?utm_source=terraform) for detailed feature guides
- Learn more about hosting OpenSource on Nexaa [OpenSource](https://nexaa.io/opensource.html?utm_source=terraform)
