# Shell Completions

Accelerate your terminal workflow by enabling Gherkio autocomplete support. Gherkio automatically outputs autocomplete scripts for Bash, Zsh, Fish, and PowerShell shells.

> [!NOTE]
> **Secure UNIX Design**: For security and transparency, Gherkio **never** mutates your shell profiles or system files automatically. The `gherkio completion` command strictly outputs the generated autocompletion script directly to standard output (`stdout`). Sourcing (`source`) or redirecting (`>`) the script gives you complete, explicit control over your terminal environment changes.

---

## ⚡ Generating Autocompletes

Generate the completion script matching your active shell using the `gherkio completion` command:

```bash
# Bash
gherkio completion bash

# Zsh
gherkio completion zsh

# Fish
gherkio completion fish

# PowerShell
gherkio completion powershell
```

---

## 🔌 Installing Completions Permanently

### 1. Zsh (macOS & Linux)

If shell completions are not already enabled in your environment, add this to your `~/.zshrc`:
```zsh
autoload -U compinit; compinit
```

Generate the Gherkio completion script and place it inside a directory listed in your `$fpath`:
```bash
gherkio completion zsh > "${fpath[1]}/_gherkio"
```

Alternatively, source it directly in your `~/.zshrc`:
```zsh
source <(gherkio completion zsh)
```

---

### 2. Bash (Linux)

Ensure `bash-completion` is installed using your package manager (e.g. `apt-get install bash-completion` or `yum install bash-completion`).

Write the completion script to the system global configuration:
```bash
gherkio completion bash | sudo tee /etc/bash_completion.d/gherkio > /dev/null
```

---

### 3. Fish

Write the completion script directly to your local user config folder:
```bash
gherkio completion fish > ~/.config/fish/completions/gherkio.fish
```

---

## ⚙️ Options

### Disable Descriptions
By default, Gherkio completions include flag and subcommand descriptions. If you prefer a faster, minimalistic autocomplete output, you can disable descriptions using the `--no-descriptions` flag:
```bash
gherkio completion zsh --no-descriptions > "${fpath[1]}/_gherkio"
```
