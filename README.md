<!-- markdownlint-disable first-line-h1 no-inline-html -->
<a href="https://terraform.io">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset=".github/terraform_logo_dark.svg">
    <source media="(prefers-color-scheme: light)" srcset=".github/terraform_logo_light.svg">
    <img src=".github/terraform_logo_light.svg" alt="Terraform logo" title="Terraform" align="right" height="50">
  </picture>
</a>

# Terraform Nexaa Provider

<!-- [![Forums][discuss-badge]][discuss]

[discuss-badge]: https://img.shields.io/badge/discuss-terraform--aws-623CE4.svg?style=flat
[discuss]: https://discuss.hashicorp.com/c/terraform-providers/tf-aws/ -->

The [Nexaa Provider]() enables [Terraform](https://terraform.io) to manage [Nexaa](cloud.tilaa.com) resources.

To start using the Nexaa provider for terraform you need to have an API account for Nexaa. See below an example for a base terraform file without any resources.

```bash
terraform {
  required_providers {
    nexaa = {
      source  = "registry.terraform.io/tilaa/nexaa"
      version = "0.1.0"
    }
  }
}

provider "nexaa" {
  username = "example@tilaa.com"
  password = "example"
}
```


Important links:
- [Tilaa knowledge base](https://support.tilaa.com)
- [Tilaa support](https://tilaa.com/support)

_**Please note:** We take Terraform's security and our users' trust very seriously. If you believe you have found a security issue in the Terraform Nexaa Provider, please responsibly disclose it by contacting us at security@hashicorp.com or support@tilaa.com._