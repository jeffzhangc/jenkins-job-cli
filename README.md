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

After running a job, you'll be prompted to save the command with an alias. You can also save commands manually by running them normally - the tool will ask if you want to save them.

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
