# p-switch

A beautiful TUI to switch between Git profiles (personal/work) on macOS.

## Features

- Pretty terminal UI with Charm Bracelet
- Keeps your emails private (config.yaml is gitignored)
- Switches both SSH config and Git user config
- Easy keyboard navigation

## Setup

### Option 1: Install from GitHub (Recommended)

```bash
go install github.com/lutefd/p-switch@latest
```

Then run `p-switch` once to generate the config file at `~/.config/p-switch/config.yaml`, edit it with your details, and run again!

### Option 2: Build from source

```bash
git clone https://github.com/lutefd/p-switch.git
cd p-switch
go build -o p-switch
chmod +x p-switch
sudo ln -sf $(pwd)/p-switch /usr/local/bin/p-switch
```

### Config Setup

On first run, p-switch will create `~/.config/p-switch/config.yaml` for you. Edit it with your details:

```yaml
profiles:
  personal:
    name: "Your Name"
    email: "your.personal@email.com"
    sshIdentityFile: "~/.ssh/personal"
  work:
    name: "Your Name"
    email: "your.work@email.com"
    sshIdentityFile: "~/.ssh/work"

sshConfigPath: "~/.ssh/config"
```

## Usage

### Switch profiles
```bash
p-switch
```
Use arrow keys or j/k to navigate, Enter to select, q to quit.

### Edit config
```bash
p-switch --edit    # Opens config in $EDITOR (defaults to vim)
```

### Find config location
```bash
p-switch --config  # Prints: ~/.config/p-switch/config.yaml
```

Or just edit directly:
```bash
vim ~/.config/p-switch/config.yaml
```

## How it works

When you select a profile, p-switch will:
1. Update your `~/.ssh/config` to use the correct IdentityFile for `github.com`
2. Update your global git config for `user.email` and `user.name`
