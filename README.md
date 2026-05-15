# svlacl

List ACLs applied to SVI (Switched Virtual Interface) interfaces from Cisco config files.

## Description

`svlacl` is a Go CLI tool that parses a single Cisco IOS/NX-OS configuration file and extracts all SVI (`interface Vlan*`) information, including:

- VLAN name
- IP address (with CIDR prefix)
- VRF assignment
- Interface status (up/down)
- Inbound and outbound ACL names

## Installation

### From source

```bash
go install github.com/ales999/svlacl@latest
```

### Build locally

```bash
git clone https://github.com/ales999/svlacl.git
cd svlacl
go build
```

A precompiled binary (`svlacl.exe`) is also included in the repository for Windows.

## Usage

```
svlacl [flags] CONFIG
```

### Arguments

| Argument   | Description                            |
|------------|----------------------------------------|
| `CONFIG`   | Cisco config file name or path         |

### Flags

| Flag | Short | Environment | Description                           |
|------|-------|-------------|---------------------------------------|
| `--config-dir` | - | `CISCONFS`  | Path to directory containing Cisco config files (required) |
| `-d` | `--debug` | — | Enable debug output                  |
| `-q` | `--quiet` | — | Lite mode — one ACL name per line (active SVI only) |
| `--unique-acls` | - | — | Remove duplicate ACL names (only with `-q`) |
| `--exclude-acls-file` | - | — | Path to a file with VLAN names to exclude from `-q` output (one per line, lowercase) |

### Examples

**Full table output:**

```bash
svlacl my-switch.cfg
```

Output:
```
Hostname: my-switch
VlanName       | IP:                        | VRF      | Status: ACL In:    | ACL Out:
vlan2006       | 172.24.2006.1/24           |          | up       | acl_in         | acl_out
vlan3001       | 10.0.3001.1/24             | MGMT     | DOWN     |                |
```

**Quiet mode (active SVIs only, one ACL per line):**

```bash
svlacl -q --config-dir /backups/cisco my-switch.cfg
```

Output:
```
acl_in
acl_out
another_acl
```

**Quiet mode with deduplication:**

```bash
svlacl -q --unique-acls --config-dir /backups/cisco my-switch.cfg
```

Output (sorted, unique ACL names only):
```
acl_in
acl_out
another_acl
```

**Quiet mode with exclusions:**

Create a file `exclude.txt` with VLAN names to exclude (one per line):
```
vlan700
vlan713
vlan933
```

```bash
svlacl -q --exclude-acls-file exclude.txt --config-dir /backups/cisco my-switch.cfg
```

SVIs matching the listed VLAN names are skipped entirely, so their ACLs won't appear in output.

**Quiet mode with deduplication and exclusions:**

```bash
svlacl -q --unique-acls --exclude-acls-file exclude.txt --config-dir /backups/cisco my-switch.cfg
```

**Debug mode with absolute path:**

```bash
svlacl -d /backups/cisco/switch-core-20240101.cfg
```

## Output format

### Default (table)

Each line shows the details of one SVI interface found in the config file:

| Column    | Description                            |
|-----------|----------------------------------------|
| VlanName  | Interface name (e.g., `Vlan2006`)      |
| IP        | Assigned IP address with prefix length |
| VRF       | VRF instance, if configured            |
| Status    | `up` or `DOWN` (based on `shutdown`)   |
| ACL In    | Inbound access-group name              |
| ACL Out   | Outbound access-group name             |

### Quiet (`-q`)

Prints only the names of ACLs bound to active (non-shutdown) SVI interfaces, one per line. Use `--unique-acls` to deduplicate and sort the output alphabetically. Use `--exclude-acls-file` to skip entire VLAN interfaces listed in an external file — those SVIs and their ACLs will not appear in the output. Comparison is case-insensitive (the exclude file uses lowercase, e.g. `vlan700`, matching `Vlan700` in the config). This mode is useful for scripting and automation.

## Parsed information

For each `interface Vlan*` block in the config file, `svlacl` extracts:

- **Hostname** — from the `hostname` directive at the top of the file
- **VLAN name** — e.g., `Vlan2006`
- **IP address** — converted to CIDR notation (e.g., `172.24.2006.1/24`)
- **VRF** — from `vrf forwarding` or `ip vrf forwarding`
- **Shutdown status** — detected by the presence of `shutdown` (excluding lines inside `description`)
- **ACL inbound/outbound** — from `ip access-group <name> in|out`

## Requirements

- Go 1.26+
- Cisco IOS/NX-OS configuration file (plain text)

## License

MIT
