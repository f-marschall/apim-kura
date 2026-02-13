# APIM-Kura

Kura is a command-line tool for backing up and restoring subscription keys from Azure API Management (APIM) instances.

Azure API Management subscription keys are critical credentials that grant access to your APIs. When infrastructure is reprovisioned, migrated, or recovered from failure, these keys are regenerated with new values. Any external system, partner, or client that relies on the original keys will break. Kura solves this by exporting subscription keys (including their secret values) to local JSON files and restoring them to any APIM instance, preserving the exact key material across environments.

## Table of Contents

- [Prerequisites](#prerequisites)
- [Installation](#installation)
- [Authentication](#authentication)
- [Commands](#commands)
  - [backup](#backup)
  - [restore](#restore)
  - [list](#list)
  - [compare](#compare)
  - [delete](#delete)
  - [clean](#clean)
- [Backup Storage Layout](#backup-storage-layout)
- [Typical Workflow](#typical-workflow)

## Prerequisites

- Go 1.23 or later
- Azure CLI (`az`) installed and authenticated
- Sufficient permissions on the target APIM instance to read and write subscriptions

## Installation

### Pre-built binaries

Download the latest release from the [GitHub Releases](https://github.com/f-marschall/apim-kura/releases) page. Binaries are available for Linux, macOS, and Windows across multiple architectures.

For example, on Linux (amd64):

```bash
curl -Lo kura https://github.com/f-marschall/apim-kura/releases/latest/download/kura-linux-amd64
chmod +x kura
sudo mv kura /usr/local/bin/
```

### Build from source

```bash
git clone https://github.com/f-marschall/apim-kura.git
cd apim-kura
go build -o kura .
```

## Authentication

Kura authenticates using your existing Azure CLI session. Before running any command, ensure you are logged in:

```bash
az login
az account set --subscription <your-subscription-id>
```

If you do not provide a `--subscription` flag to a command, Kura resolves the subscription ID automatically from the currently active Azure CLI account.

## Commands

### backup

```
kura backup --resource-group <rg> --apim-name <apim> [--product-id <product>] [--subscription <sub-id>]
```

The backup command connects to an Azure API Management instance, retrieves every subscription key (including primary and secondary secret values), and writes them to a local JSON file.

The philosophy behind backup is non-destructive, read-only access. It does not modify anything in Azure. It captures the full subscription contract -- display name, state, scope, owner, tracing configuration, timestamps, and both secret keys -- so that a restore can reproduce the subscription exactly as it was.

When `--product-id` is provided, the backup is scoped to only those subscriptions associated with that specific product. This is useful when you manage many products and want targeted, smaller backup files rather than a single monolithic export.

| Flag | Short | Required | Description |
|------|-------|----------|-------------|
| `--resource-group` | `-g` | Yes | Azure resource group containing the APIM instance |
| `--apim-name` | `-a` | Yes | Name of the APIM instance |
| `--product-id` | `-p` | No | Scope backup to a single product |
| `--subscription` | `-s` | No | Azure subscription ID (defaults to current CLI context) |

### restore

```
kura restore --resource-group <rg> --apim-name <apim> --input <file> [--subscription <sub-id>] [--dry-run]
```

The restore command reads a previously created backup file and recreates each subscription in the target APIM instance. It uses the Azure `CreateOrUpdate` API, meaning it will create subscriptions that do not exist and overwrite those that do. The original subscription ID (GUID), display name, keys, state, owner, and tracing settings are all preserved.

The philosophy behind restore is idempotent, environment-agnostic replay. A backup taken from one APIM instance can be restored to a different instance (even in a different resource group or Azure subscription). Kura extracts the product ID from the scope stored in the backup and rebuilds the full resource path against the target environment. This makes it suitable for disaster recovery, environment promotion, and infrastructure-as-code workflows where APIM is torn down and recreated.

**⚠️ Warning:** The `master` subscription is a built-in system subscription that cannot be recreated. It is automatically skipped during restore operations and its keys are not restored.

The `--dry-run` flag previews every subscription that would be created or updated without making any changes. This is intended for validation before committing to a restore operation.

| Flag | Short | Required | Description |
|------|-------|----------|-------------|
| `--resource-group` | `-g` | Yes | Target Azure resource group |
| `--apim-name` | `-a` | Yes | Target APIM instance name |
| `--input` | `-i` | Yes | Path to the backup JSON file |
| `--subscription` | `-s` | No | Azure subscription ID (defaults to current CLI context) |
| `--dry-run` | | No | Preview changes without applying them |

### list

```
kura list --resource-group <rg> --apim-name <apim> [--product-id <product>] [--subscription <sub-id>]
```

The list command is a diagnostic and inspection tool. It connects to an APIM instance and prints every subscription key to the terminal in a human-readable format. Unlike backup, it does not write anything to disk. Its purpose is to give operators a quick view of current subscription state -- useful for auditing, verifying a restore, or comparing keys across environments.

When `--product-id` is provided, the output is filtered to subscriptions scoped to that product.

| Flag | Short | Required | Description |
|------|-------|----------|-------------|
| `--resource-group` | `-g` | Yes | Azure resource group containing the APIM instance |
| `--apim-name` | `-a` | Yes | Name of the APIM instance |
| `--product-id` | `-p` | No | Filter output to a single product |
| `--subscription` | `-s` | No | Azure subscription ID (defaults to current CLI context) |

### compare

```
kura compare <file1> <file2>
```

The compare command reads two backup JSON files and displays the differences between them. Use this to audit changes, verify backup consistency, or compare subscription keys across different snapshots.

### delete

```
kura delete --resource-group <rg> --apim-name <apim> --subscription-id <id> [--subscription <sub-id>]
```

The delete command removes a subscription from an APIM instance. Specify the subscription ID (GUID) to delete.

| Flag | Short | Required | Description |
|------|-------|----------|----------|
| `--resource-group` | `-g` | Yes | Azure resource group containing the APIM instance |
| `--apim-name` | `-a` | Yes | Name of the APIM instance |
| `--subscription-id` | `-i` | Yes | The subscription ID (GUID) to delete |
| `--subscription` | `-s` | No | Azure subscription ID (defaults to current CLI context) |

### clean

```
kura clean
```

The clean command removes the entire local `backup/` directory and all of its contents. It takes no flags. Its purpose is housekeeping -- after a restore is verified, or when backup data is no longer needed, clean provides a single command to remove all locally stored secrets rather than requiring manual deletion. This is important because backup files contain plaintext subscription keys and should not persist on disk longer than necessary.

## Backup Storage Layout

All backups are written under a `backup/` directory relative to the current working directory. The directory structure encodes the resource group, APIM instance name, and optionally the product ID:

```
backup/
  <resource-group>/
    <apim-name>/
      subscriptions.json          # Full instance backup
      <product-id>/
        subscriptions.json        # Product-scoped backup
```

For example, running:

```bash
kura backup -g apim-kura -a gh-apim-kura-main
kura backup -g apim-kura -a gh-apim-kura-main -p sample-product-2
```

Produces:

```
backup/
  apim-kura/
    gh-apim-kura-main/
      subscriptions.json
      sample-product-2/
        subscriptions.json
```

Each `subscriptions.json` file is a JSON array of subscription objects containing the full subscription contract including both primary and secondary keys.

**Important:** These files contain sensitive credentials. Do not commit them to version control. Add `backup/` to your `.gitignore`.

## Typical Workflow

1. **Backup** subscription keys from the source APIM instance:
   ```bash
   kura backup -g my-resource-group -a my-apim
   ```

2. **Verify** the backup by inspecting the JSON file or listing the current state:
   ```bash
   kura list -g my-resource-group -a my-apim
   ```

3. **Preview** the restore against the target environment:
   ```bash
   kura restore -g target-rg -a target-apim -i backup/my-resource-group/my-apim/subscriptions.json --dry-run
   ```

4. **Restore** the subscription keys:
   ```bash
   kura restore -g target-rg -a target-apim -i backup/my-resource-group/my-apim/subscriptions.json
   ```

5. **Clean up** local backup files once the restore is confirmed:
   ```bash
   kura clean
   ```