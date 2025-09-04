---
page_title: "Nexaa Resources"
description: "The resources the terraform provider provides"
---

# Resources
## Introduction
This is the introduction about the resources we are offering. We start by defining a namespace where our project will live
and share resources like Registries, Persistent Volumes and more. Namespaces have only a required field which is their unique name
and an optional description that help us provide more context if needed.

## Registry (nexaa_registry)
Container images live online. This could be public hubs, like Docker Hub, or even a private registry where authentication is needed.
That's why we provide a way to log in to your private registry which then will be used when the container image needs to be pulled from. <br>
Tip: Are you using an image from a public repository ? Then there's no reason to specify a registry

### Links
- Learn more about [registries][docs_registry] in our documentation.
- Configuring [Registry](../resources/registry.md)

## Volumes (nexaa_volume)
Your data are persistent and shared within a Nexaa namespace. Every container can make use of those volumes as mounts when created or configured. <br>
You can always scale your storage needs up if you see that you are in need of more space for your application.

### Links
- Learn more about [volumes][docs_volumes] in our documentation.
- Configuring [Volumes](../resources/volume.md)

## Container (nexaa_container)
A container is an encapsulated environment with your application running in it. You can deploy your container on nexaa by only specifying your image. When this image is hosted on a private registry you have to add you [registry](#registry-(nexaa_registry)) to the configuration.
Nexaa is supporting all the powerful perks a container is packed with such as:
### Scaling 
- **Auto-scaling** with minimum and maximum number of replicas and replication triggers
- **Manual scaling** where you predefine the number of replicas that your container will run.

### Ingress
Expose your application to the internet by defining your Domain name, the port, the use of TLS and an ip allow-list. Pro tip: You get unmetered traffic ! 

### Environment variables
You can define your environment variables and mark them as secrets if needed. Nexaa is saving those secret values in a vault, and we have no access on them.

### Health check
You can access your container through an HTTP path on a specific port, where you can perform actions like rolling updates and health checks. <br>
To have rolling updates you can specify an http health check on your container. A new container will be started before the old one will be killed. Tip: Best practice is to have more than 1 container when doing rolling updates.


### Links
- Learn more about [containers][docs_container] in our documentation.
- Configuring [Containers](../resources/container.md)

## Container Job (nexaa_container_job)
A container job is a container which runs a task or script at a specified time. 
This can for example once a week, once a month and more as long it can be written as cron notation. 
The command and entrypoint can be overwritten in the configuration, so you can use 1 container for more actions.
```text
  * * * * *
# | | | | |
# | | | | day of the week (0–6) (Sunday to Saturday) 
# | | | month (1–12)             
# | | day of the month (1–31)
# | hour (0–23)
# minute (0–59)
```
### Links
- Learn more about [container jobs][docs_container_job] in our documentation.
- Configuring [Container Jobs](../resources/container_job.md)

## Cloud Database Cluster (nexaa_cloud_database_cluster)
We provide Cloud Database Clusters which is more than a database. It is a whole cluster. You can specify how many replicas you want
can be 1 up to 3 replicas per cluster. This means that your data will be replicated x amount of times. To learn more about this you can read on our [documentation page][docs_cloud_database_cluster]].

### Links
- Learn more about [Cloud database cluster][docs_cloud_database_cluster] in our documentation.
- Configuring [Cloud Database Cluster](../resources/cloud_database_cluster.md)

### Database (nexaa_cloud_database_cluster_database)
After successfully creation of a cloud database cluster, we can create databases inside it. You must provide a name for your database and optionally a description for it. 
You are not charged per database inside your cluster, so feel free experimenting with up to the limits of your selected database cluster engine.


### Links
- Configuring [Database](../resources/cloud_database_cluster_database.md) on our cloud database clusters

### Users (nexaa_cloud_database_cluster_user)
After Creating a database you might want to have a specific user to access that database. You can specify the permissions on the database which is read only or read write.
You can change this whenever you want, you can also add a user to multiple databases.

### Links
- Configuring [User](../resources/cloud_database_cluster_user.md) on our cloud database clusters


## What's next
- Documentation about our platform [Nexaa documentation][docs]

## Need Help?
Visit the [Nexaa documentation][docs] for detailed guides and tutorials on specific features and use cases.

[docs]: https://docs.nexaa.io/?utm_source=terraform
[docs_container]: https://docs.nexaa.io/serverless-containers/containers/?utm_source=terraform
[docs_container_job]: https://docs.nexaa.io/serverless-containers/container-jobs/?utm_source=terraform
[docs_volumes]: https://docs.nexaa.io/serverless-containers/persistent-storage/?utm_source=terraform
[docs_registry]: https://docs.nexaa.io/serverless-containers/registries/?utm_source=terraform
[docs_cloud_database_cluster]: https://docs.nexaa.io/cloud-databases/introduction/?utm_source=terraform
