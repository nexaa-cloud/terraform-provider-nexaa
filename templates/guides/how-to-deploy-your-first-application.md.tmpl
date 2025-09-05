---
page_title: "Deploy your first application"
description: "How to deploy your first application on Nexaa"
subcategory: "How to"
---

# Deploy your first application
We will show you have to deploy your first application.


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
- **Container image**
To deploy your first application, you will need a container image hosted on a repository. You can also use publicly accessible images, like the ones you can find on [Docker hub](https://hub.docker.com).

***

## Namespace
Before we start you will need a namespace, this is where your application will be living.
```terraform
resource "nexaa_namespace" "namespace" {
  name        = "namespace-name"
  description = "" # Description is optional
}
```

## Application
Before we can start we need to know what your application needs. For example, a contianer, a database, persistent storage or a registry if your container image is hosted on a private repository.
We will go through this step by step, at the end you will find the full example. 

### Persistent storage?
Does your application need shared and persistent storage among the containers? Then it's the perfect time to specify a nexaa_volume. This will create a volume which can be mounted on a path in container(s). If mounted to multiple containers multiple containers can write and read to it.
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

### Database?
Does your application need to store structured data? Then you can make use of our cloud database clusters. This can be a cluster up to 3 replicas.

> **Note:** the cloud database clusters need a data source, this data source validates the resources you want to give to your cluster.
```terraform
data "nexaa_cloud_database_cluster_plans" "plan" {
  cpu      = 1
  memory   = 2.0
  storage  = 10
  replicas = 1
}
```

```terraform
variable "myfirstuserpassword" {
  sensitive = true
  type      = string
}

resource "nexaa_cloud_database_cluster" "cluster" {
  depends_on = [nexaa_namespace.project]
  cluster = {
    name      = "myfirstcluster"
    namespace = nexaa_namespace.project.name
  }

  spec = {
    type    = "PostgreSQL" # type can also be MySQL with the correct version
    version = "16.4"
  }

  plan = data.nexaa_cloud_database_cluster_plans.plan.id
}

resource "nexaa_cloud_database_cluster_database" "mydatabase" {
  depends_on  = [nexaa_cloud_database_cluster.cluster]
  name        = "mydatabase"
  description = ""
  cluster     = nexaa_cloud_database_cluster.cluster.cluster
}

resource "nexaa_cloud_database_cluster_user" "user" {
  depends_on = [
    nexaa_cloud_database_cluster.cluster,
  ]

  cluster  = nexaa_cloud_database_cluster.cluster.cluster
  name     = "myfirstuser"
  password = var.myfirstuserpassword
  permissions = [
    {
      database_name = nexaa_cloud_database_cluster_database.db1.name,
      permission    = "read_write",
      state         = "present",
    }
  ]
}

```

### Registry
If your image on a private registry, you will nee to add a registry. This provides us a way to pull your image, so we can run it.

```terraform
variable "registry_password" {
  sensitive = true
  type = string
}

resource "nexaa_registry" "registry" {
  namespace = nexaa_namespace.namespace.name
  name      = "private"
  source    = "somewhere.safe.example.com"
  username  = "user"
  password  = var.registry_password
  verify    = true
}
```

### Container
After our preparation, we are ready to deploy our application. This is where we would have to specify the Container image. <br>
We prepared two examples for you, which you can choose from depending on your needs; one with a volume and one without.

> **Note:** the containers and container_jobs need a data source, this data source validates the resources you want to give to your cluster.

```terraform
data "nexaa_container_resources" "container_resource" {
  cpu = 0.25
  memory = 0.5
}
```

#### Container without a volume
```terraform
resource "nexaa_container" "container" {
  depends_on = [
    nexaa_namespace.namespace
  ]
  name      = "container"
  namespace = nexaa_namespace.namespace.name
  image     = "mycontainerimage:latest"
  
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

#### Container with a volume
```terraform
resource "nexaa_container" "container" {
  depends_on = [
    nexaa_namespace.namespace,
    nexaa_volume.volume
  ]
  name      = "container"
  namespace = nexaa_namespace.namespace.name
  image     = "mycontainerimage:latest"
  
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

### Container Job
To do recurring tasks as processing images or clearing cache, we recommend using container_jobs. 
This means a certain task will be ran every x amount of time. 

```terraform
resource "nexaa_container_job" "containerjob" {
  name      = "myrecurringjob"
  namespace = nexaa_namespace.namespace.name
  image     = "mycontainerimage:latest"
  registry  = nexaa_registry.registry.name # Can be excluded if your image is public

  resources = data.nexaa_container_resources.container_resource.id

  mounts = [
    {
      path   = "/storage/mount"
      volume = nexaa_volume.volume.name
    }
  ]

  schedule_cron_expression = "0 * * * *" # Every hour
  command                  = ["/bin/sh", "-c", "date; echo Hello World"]
}
```

## Full example
Here is a full example of an application

```terraform
# ===================================================================
# Variables
# ===================================================================


variable "myfirstuserpassword" {
  sensitive = true
  type      = string
}

# ===================================================================
# Data sources
# ===================================================================

data "nexaa_cloud_database_cluster_plans" "plan" {
  cpu      = 1
  memory   = 2.0
  storage  = 10
  replicas = 1
}


data "nexaa_container_resources" "container_resource" {
  cpu = 0.25
  memory = 0.5
}

# ===================================================================
# Resources
# ===================================================================

resource "nexaa_namespace" "namespace" {
  name        = "namespace-name"
  description = "" # Description is optional
}

resource "nexaa_volume" "volume" {
  depends_on = [
    nexaa_namespace.namespace,
  ]
  namespace = nexaa_namespace.namespace
  name      = "volume"
  size      = 1 # the size of the volume can be 1 - 100, keep in mind the size can only grow not shrink.
}

resource "nexaa_cloud_database_cluster" "cluster" {
  depends_on = [nexaa_namespace.project]
  cluster = {
    name      = "myfirstcluster"
    namespace = nexaa_namespace.project.name
  }

  spec = {
    type    = "PostgreSQL" # type can also be MySQL with the correct version
    version = "16.4"
  }

  plan = data.nexaa_cloud_database_cluster_plans.plan.id
}

resource "nexaa_cloud_database_cluster_database" "mydatabase" {
  depends_on  = [nexaa_cloud_database_cluster.cluster]
  name        = "mydatabase"
  description = ""
  cluster     = nexaa_cloud_database_cluster.cluster.cluster
}

resource "nexaa_cloud_database_cluster_user" "user" {
  depends_on = [
    nexaa_cloud_database_cluster.cluster,
  ]

  cluster  = nexaa_cloud_database_cluster.cluster.cluster
  name     = "myfirstuser"
  password = var.myfirstuserpassword
  permissions = [
    {
      database_name = nexaa_cloud_database_cluster_database.db1.name,
      permission    = "read_write",
      state         = "present",
    }
  ]
}

resource "nexaa_container" "container" {
  depends_on = [
    nexaa_namespace.namespace,
    nexaa_volume.volume
  ]
  name      = "container"
  namespace = nexaa_namespace.namespace.name
  image     = "mycontainerimage:latest"

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

resource "nexaa_container_job" "containerjob" {
  name      = "myrecurringjob"
  namespace = nexaa_namespace.namespace.name
  image     = "mycontainerimage:latest"
  registry  = nexaa_registry.registry.name # Can be excluded if your image is public

  resources = data.nexaa_container_resources.container_resource.id

  mounts = [
    {
      path   = "/storage/mount"
      volume = nexaa_volume.volume.name
    }
  ]

  schedule_cron_expression = "0 * * * *" # Every hour
  command                  = ["/bin/sh", "-c", "date; echo Hello World"]
}
```


## Next Steps
- Explore our [marketplace](marketplace.md)
- Explore the [complete documentation](https://docs.nexaa.io/?utm_source=terraform) for detailed feature guides
- Learn more about hosting OpenSource on Nexaa [OpenSource](https://nexaa.io/opensource.html?utm_source=terraform)
- Learn more about the [resources](nexaa-resources.md) we provide 
