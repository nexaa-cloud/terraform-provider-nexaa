# NOTE: This file is for HashiCorp specific licensing automation and can be deleted after creating a new repo with this template.
schema_version = 1

project {
  license          = "MPL-2.0"
  copyright_year   = 2026
  copyright_holder = "Tilaa B.V."

  header_ignore = [
    # examples used within documentation (prose)
    "examples/**",
    # GitHub issue template configuration
    ".github/ISSUE_TEMPLATE/*.yml",
    # golangci-lint tooling configuration
    ".golangci.yml",
    # GoReleaser tooling configuration
    ".goreleaser.yml",
    "*.sh",
    "*.md"
  ]
}