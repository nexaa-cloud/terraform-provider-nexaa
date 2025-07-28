<!-- markdownlint-disable first-line-h1 no-inline-html -->
<a href="https://nexaa.io?utm_source=github&utm_campaign=terraform">
  <picture>
    <img src="https://nexaa.io/assets/nexaa-logo.svg?utm_source=github&utm_campaign=terraform" alt="Terraform logo" title="Terraform" align="right" height="50">
  </picture>
</a>

# Nexaa Terraform Provider

The [Nexaa Provider](https://github.com/nexaa-cloud/terraform-provider-nexaa/) enables [Terraform](https://terraform.io) to manage [Nexaa](https://portal.nexaa.io?utm_source=github&utm_campaign=terraform) resources.

To start using the Nexaa provider for terraform you need to have an API account for Nexaa. See below an example for a base terraform file without any resources.

```tf
terraform {
  required_providers {
    nexaa = {
      source  = "registry.terraform.io/tilaa/nexaa"
      version = "0.1.0"
    }
  }
}

provider "nexaa" {
  username = "user@example.com"
  password = "example"
}
```

## Contributing and Developing the provider

To start contributing to the provider you need to use a local version for developing. First you need to pull the provider code from [github](http://github.com/nexaa-cloud/terraform-provider-nexaa). 

Then you need to install the binary of the provider. That can be done by using the command below. Remember that everytime you made a change in the provider and it's not deployed execute this command again to update your local binary.
```bash
go install .
```


This will also install every dependency needed in the go.mod file of the provider. Part of the dependencies is the nexaa-CLI. You can also use a local version of the CLI. To do this you need to pull the cli from [github](https://github.com/nexaa-cloud/nexaa-cli). Then in the go.mod file of the provider add this line and make it point to the go.mod file of the nexaa-CLI. then run the previous command again.
```bash
replace gitlab.com/tilaa/tilaa-cli => absolute/path/to/the/go.mod
```

The next step is to make terraform use the local binary instead of searching in the online registry. To do this create a **.terraformrc** file in the root directory of your machine and this text to it and change the path to the absolute path of the provider binary. Usually it's located in the **go/bin** folder.
```go
provider_installation 
{
	dev_overrides {
		"registry.terraform.io/tilaa/nexaa" = "absolute/path/to/provider/binary"
	}
	direct {}
}
```

Now to start using your local provider create a tf file the same way you would when using the deployed provider. The only difference is that you can immediately start using it using
```bash
terraform apply
```




Important links:
- [Tilaa knowledge base](https://support.tilaa.com)
- [Tilaa support](https://tilaa.com/support)


_**Please note:** We take Terraform's security and our users' trust very seriously. If you believe you have found a security issue in the Terraform Nexaa Provider, please responsibly disclose it by contacting us at support@tilaa.com._
