# Easy way to run Jenkins job from the Command Line

<meta name="google-site-verification" content="Wl2WZRolJ6omFNTQRguTy0GRQU41taSDq20n4Qgz05c" />

The utility starts a Jenkins build/job from the Command Line/Terminal.
An execution will be like this:

![terminal demo](assets/demo.gif)

## Install

Fetch the [latest release](https://github.com/gocruncher/jenkins-job-cli/releases) for your platform:

#### Linux

```bash
sudo wget https://github.com/gocruncher/jenkins-job-cli/releases/download/v1.1.2/jenkins-job-cli-1.1.2-linux-amd64 -O /usr/local/bin/jj
sudo chmod +x /usr/local/bin/jj
```

#### OS X brew

```bash
# brew tap gocruncher/tap
# brew install jj
brew install jeffzhangc/tap/jenkins-job-cli
```

#### OS X bash

```bash
sudo curl -Lo /usr/local/bin/jj https://github.com/gocruncher/jenkins-job-cli/releases/download/v1.1.2/jenkins-job-cli-1.1.2-darwin-amd64
sudo chmod +x /usr/local/bin/jj
```

## Getting Started

### Configure Access to Multiple Jenkins

```bash
jj set dev_jenkins --url "https://myjenkins.com" --login admin --token 11aa0926784999dab5
```

where the token is available in your personal configuration page of the Jenkins. Go to the Jenkins Web Interface and click your name on the top right corner on every page, then click "Configure" to see your API token.

In case, when Jenkins is available without authorization:

```bash
jj set dev_jenkins --url "https://myjenkins.com"
```

or just run the following command in dialog execution mode:

```bash
jj set dev_jenkins
```

### Shell autocompletion

As a recommendation, you can enable shell autocompletion for convenient work. To do this, run following:

```bash
# for zsh completion:
echo 'source <(jj completion zsh)' >>~/.zshrc

# for bash completion:
echo 'source <(jj completion bash)' >>~/.bashrc
```

if this does not work for some reason, try following command that might help you to figure out what is wrong:

```bash
jj completion check
```

### Configuration Location

`jj` stores its configuration under `$XDG_CONFIG_HOME/jj` (default
`~/.config/jj/`):

```bash
~/.config/jj/
├── config.yaml    # configured Jenkins environments + history options
├── history.yaml   # saved quick commands
└── cache.<env>    # cached job tree per environment
```

The first time a newer `jj` starts, it detects the legacy `~/.jj/`
directory. If a `config.yaml` is found there, a notice is printed and
the directory is migrated automatically (rename first, fallback to
copy when the rename crosses filesystems).

History-related knobs live in `config.yaml`:

```yaml
# Cap of non-pinned quick commands. 0 or negative resets to the default (10).
history_limit: 10

# Alias template. Empty keeps the default "<job>_<8-char-hash>".
history_alias_template: "{job}_{param.branch}"
```

### Examples

```bash
# Configure Access to the Jenkins
jj set dev-jenkins

# Start 'app-build' job in the current Jenkins
jj run app-build

# Start 'web-build' job in Jenkins named prod
jj run -n prod web-build

# makes a specific Jenkins name by default
jj use PROD

# list all running jobs
jj console

# display the console of the latest job
jj console app-xxx
```

## History Management

The `jj history` command allows you to save, manage, and quickly rerun frequently used Jenkins job commands with custom aliases. This feature helps you avoid typing long commands repeatedly.

### Save Quick Commands

Every successful `jj run` is **recorded automatically** — no prompt, no
manual alias entry. Identical invocations (same environment, job and
parameters) update the same record by incrementing `Executimes` and
refreshing `LastExecTime`. Different invocations get a new record with a
new alias.

```bash
# 1. Plain run — alias is auto-generated (see Alias Templates) and history is updated
jj run web-rpm-build-manual -n dev -a branch=main

# 2. Same command again — same alias, Executimes += 1
jj run web-rpm-build-manual -n dev -a branch=main

# 3. Different branch — a new record, a new alias
jj run web-rpm-build-manual -n dev -a branch=dev
```

You can also save a command without running it:

```bash
# Manually add a command with alias (interactive)
jj history add "jj run cc-stat3 -n aicc -a env=dev-93 -a DeliveryModel=BuildAndDeploy"
```

**Alias Duplicate Handling:**

- If you saved the command with `history add` and the alias is already in
  use, the existing record is overwritten (Pinned flag and params hash
  are preserved).
- Auto-generated aliases are unique by construction, so collisions are
  not possible there.

### Alias Templates

By default, an auto-generated alias looks like `<job>_<8-char-hash>`
(e.g. `web-rpm-build-manual_a3f9c1b2`), which is unique but hard to
read. Set `history_alias_template` in `config.yaml` to control the
naming:

```yaml
history_alias_template: "{job}_{param.branch}"
```

Supported placeholders:

| Placeholder       | Description                                            |
|-------------------|--------------------------------------------------------|
| `{job}`           | The Jenkins job name                                   |
| `{env}`           | The configured environment name                        |
| `{hash}`          | Stable 8-char hash of the parameter set                |
| `{param.<name>}`  | Value of the `-a <name>=<value>` argument              |

Missing parameters render as empty strings. The result is
auto-sanitised: shell-special characters and whitespace become `_`,
leading dashes are stripped. If the rendered alias collides with an
existing record, `jj` appends `_2`, `_3`, ... automatically. An empty
template or one that renders to an empty string falls back to
`<job>_<hash>`.

### Custom Aliases

Pass `--alias <name>` to `jj run` to override the generated alias for
that invocation:

```bash
jj run web-rpm-build-manual -a branch=main --alias deploy-prod
```

Alias rules: letters, digits, `.`, `_` and `-` only; ≤ 64 characters;
must not start with `-`. If the alias already exists with the same
(env, job, params), the record is updated. Otherwise the new alias is
appended as a separate record. To rename an existing record after the
fact, see `history rename` below.

### List Saved Commands

```bash
# List all saved quick commands
jj history list

# List commands in a specific environment
jj history list -e prod

# Limit the number of results
jj history list -l 10

# Show all details including full command
jj history list -a

# Output in different formats
jj history list -f json
jj history list -f yaml
```

Pinned records are prefixed with `*` and rendered in bold, so they
stand out from LRU-eligible records.

### Run Saved Commands

```bash
# Run a saved command by alias
jj history run myjob_quick

# Run multiple commands at once
jj history run alias1 alias2 alias3

# Force run without confirmation
jj history run -f myjob_quick
```

### View Command Details

```bash
# View details of a saved command
jj history view alias1

# View multiple commands
jj history view alias1 alias2

# Output in JSON format
jj history view -f json alias1
```

### Search Commands

```bash
# Search for commands by keyword
jj history search "prod"

# Search in a specific environment
jj history search -e dev "deploy"

# Output in JSON format
jj history search -f json "job"
```

### Delete Commands

```bash
# Delete a saved command
jj history delete alias1

# Delete multiple commands
jj history delete alias1 alias2

# Force delete without confirmation
jj history delete -f alias1
```

### Pin and Unpin Commands

History has a soft cap (default `history_limit`, see Configuration
Location). When the cap is exceeded, the **oldest non-pinned** record
is evicted (LRU). Pin a record to keep it forever:

```bash
jj history pin <alias>     # Mark as pinned (exempt from eviction)
jj history unpin <alias>   # Allow it to be evicted again
```

Pinning or unpinning an already-pinned / already-unpinned record is a
no-op. An unknown alias returns a clear error.

### Rename a Command

```bash
jj history rename <old-alias> <new-alias>
```

The new alias must be unique and follow the same character rules as
`--alias` (letters, digits, `.`, `_`, `-`; ≤ 64 characters; no leading
dash). Pinning state, params hash, execution counter and last-executed
timestamp are preserved.

### Clear All History

```bash
# Clear all saved commands (with confirmation)
jj history clear

# Force clear without confirmation
jj history clear -f
```

### Export and Import

```bash
# Export history to a file
jj history export history.json
jj history export history.yaml

# Export specific environment
jj history export -e prod prod.yaml

# Import history from a file
jj history import backup.yaml

# Merge instead of replace when importing
jj history import -m backup.yaml
```

### History Command Aliases

- `history` / `hist` / `h` - Main history command
- `list` / `ls` / `l` - List commands
- `run` / `r` / `exec` - Run commands
- `view` / `v` / `show` / `info` - View details
- `delete` / `del` / `rm` / `remove` - Delete commands
- `clear` / `clean` / `clr` - Clear all
- `search` / `s` / `find` / `grep` - Search commands
- `export` / `exp` / `save` - Export history
- `import` / `imp` / `load` - Import history
- `add` / `a` / `save` - Add command to history
- `pin` / `p` - Pin a command (exempt from eviction)
- `unpin` / `unp` - Unpin a command
- `rename` / `mv` - Rename a command's alias

## Features

- ✅ Run Jenkins jobs from command line
- ✅ Manage multiple Jenkins instances
- ✅ Save and rerun frequently used commands with aliases
- ✅ Search and manage command history
- ✅ Export/import command history
- ✅ Job cancellation (Ctrl+C key)
- ✅ Resize output (press enter key)
- ✅ Output of child jobs
- ✅ Show console info
- ✅ Shell autocompletion support

## Useful packages

- [cobra](https://github.com/spf13/cobra) - library for creating powerful modern CLI
- [chalk](https://github.com/chalk/chalk) – Terminal string styling done right
- [bar](https://github.com/superhawk610/bar) - Flexible ascii progress bar.

## Todos

- add authorization by login/pass and through the RSA key
- support of a terminal window resizing

## Similar projects

- [jcli](https://github.com/jenkins-zh/jenkins-cli/) was written by Golang which can manage multiple Jenkins
- [jenni](https://github.com/m-sureshraj/jenni)

## License

`jenkins-job-cli` is open-sourced software licensed under the [MIT](LICENSE) license.
