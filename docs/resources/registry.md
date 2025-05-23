---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "nexaa_registry Resource - nexaa"
subcategory: ""
description: |-
  
---

# nexaa_registry (Resource)





<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `name` (String) The name given to the private registry
- `namespace` (String) Name of the namespace the private registry belongs to
- `password` (String, Sensitive) The password used to connect to the source
- `source` (String) The URL of the site where the credentials are used
- `username` (String) The username used to connect to the source

### Optional

- `verify` (Boolean) If true(default) the connection will be tested immediately to check if the credentials are true

### Read-Only

- `id` (String) Identifier of the private registry, equal to the name of the registry
- `last_updated` (String) Timestamp of the last Terraform update of the private registry
- `locked` (Boolean) If the registry is locked it can't be deleted
