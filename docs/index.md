---
page_title: "resend Provider"
description: |-
  The Resend provider manages Resend sending domains and API keys with Terraform or OpenTofu.
---

# resend Provider

The Resend provider manages Resend sending domains and API keys with Terraform or OpenTofu.

It covers the following, with [Resend's documentation](https://resend.com/docs) as the reference for the concepts:

- `resend_domain`: add a sending domain and read the DNS records Resend generates for it.
- `resend_domain_verification`: ask Resend to verify a domain once its records are published.
- `resend_api_key`: create API keys, optionally limited to sending from one domain.
- The `resend_domain` and `resend_api_key` data sources: read existing domains and API keys by ID.

~> **Not published yet.** The provider will be published to the Terraform Registry and the
OpenTofu Registry at v0.1.0. Until then, install it from source as described under
[Installation](#installation).

## Example Usage

```terraform
terraform {
  required_providers {
    resend = {
      source  = "SimplicityGuy/resend"
      version = "~> 0.1"
    }
  }
}

# The API key can come from the RESEND_API_KEY environment variable instead;
# omit api_key from the block to use it.
provider "resend" {
  api_key = var.resend_api_key
}

variable "resend_api_key" {
  description = "A Resend API key with full_access permission."
  type        = string
  sensitive   = true
}
```

## Authentication

The provider authenticates with a [Resend API key](https://resend.com/docs/dashboard/api-keys/introduction).
Managing domains and API keys needs a key with `full_access` permission; a `sending_access`
key is rejected for those calls.

Set the key in one of two ways. The `api_key` argument wins when both are set.

1. The `RESEND_API_KEY` environment variable (recommended, keeps the key out of your
   configuration):

   ```shell
   export RESEND_API_KEY="re_..."
   ```

   ```terraform
   provider "resend" {}
   ```

2. The `api_key` argument, typically from a sensitive variable as in the example above.
   Don't commit the key to version control.

## Supported Terraform and OpenTofu versions

The acceptance suite and the examples run in CI against the oldest and the latest version of each
CLI:

| CLI       | Oldest supported | Latest tested |
| --------- | ---------------- | ------------- |
| Terraform | `1.0.11`         | `1.17.0`      |
| OpenTofu  | `1.6.3`          | `1.13.2`      |

The provider speaks plugin protocol 6, which needs Terraform 1.0 or later, and OpenTofu's first
release is 1.6. Versions in between are expected to work but are not tested separately. Older
versions are not supported.

## Installation

The provider works with both Terraform and OpenTofu. Declare it in `required_providers`:

```terraform
terraform {
  required_providers {
    resend = {
      source  = "SimplicityGuy/resend"
      version = "~> 0.1"
    }
  }
}
```

The same source address works with both CLIs: Terraform resolves it to
`registry.terraform.io/simplicityguy/resend` and OpenTofu to
`registry.opentofu.org/simplicityguy/resend`.

### Terraform

Once v0.1.0 is published, `terraform init` installs the provider from the
[Terraform Registry](https://registry.terraform.io/providers/SimplicityGuy/resend/latest).

### OpenTofu

Once v0.1.0 is published, `tofu init` installs the provider from the
[OpenTofu Registry](https://search.opentofu.org/provider/simplicityguy/resend/latest).

### From source, until v0.1.0 is published

Build the provider into a local filesystem mirror and tell your CLI to install
`SimplicityGuy/resend` from it. This works the same way for Terraform and OpenTofu, and
`init` keeps installing every other provider from the registry. It needs Go (the version
is pinned in the repository's `mise.toml`).

```shell
git clone https://github.com/SimplicityGuy/terraform-provider-resend.git
cd terraform-provider-resend

mirror="$HOME/.terraform.d/resend-mirror"
target="$(go env GOOS)_$(go env GOARCH)"
# The mirror is laid out by registry host, so add the host your CLI uses (or both).
for host in registry.terraform.io registry.opentofu.org; do
  dir="$mirror/$host/simplicityguy/resend/0.1.0/$target"
  mkdir -p "$dir"
  go build -o "$dir/terraform-provider-resend_v0.1.0" .
done
```

The build is offered as version 0.1.0, which satisfies `version = "~> 0.1"`. Then add the
mirror to your CLI configuration. Terraform reads `~/.terraformrc` (or the file named by
`TF_CLI_CONFIG_FILE`); OpenTofu reads `~/.tofurc` (or the file named by
`TOFU_CLI_CONFIG_FILE`, falling back to `TF_CLI_CONFIG_FILE`):

```terraform
provider_installation {
  filesystem_mirror {
    path    = "/home/you/.terraform.d/resend-mirror"
    include = ["simplicityguy/resend"]
  }

  # Install every other provider from the registry as usual.
  direct {
    exclude = ["simplicityguy/resend"]
  }
}
```

Run `terraform init` or `tofu init` as usual. After rebuilding, delete
`.terraform.lock.hcl` (or run `init -upgrade`) so the lock file accepts the new binary.
Remove the mirror once you switch to the published provider.

-> **Developing the provider itself?** A `dev_overrides` block pointing at the directory
holding the binary (`"simplicityguy/resend" = "/home/you/go/bin"` after `go install .`)
skips `init` for this provider and ignores version constraints. With OpenTofu, `tofu init`
fails while an override names a provider that no registry has, so use the mirror if your
configuration also needs other providers.

<!-- schema generated by tfplugindocs -->
## Schema

### Optional

- `api_key` (String, Sensitive) The Resend API key the provider authenticates with. It needs `full_access` permission to manage domains and API keys. Falls back to the `RESEND_API_KEY` environment variable; one of the two is required.
