# Gosh

**Gosh** is a lightweight, POSIX-compliant command-line shell implementation written in Go. It mimics the behavior of standard shells like Bash or Zsh, supporting built-in commands, external process execution, robust argument parsing, and I/O redirection.

## Features

### Built-in Commands
Gosh supports core shell built-ins natively:
- **`cd`**: Change the current working directory (supports `~` and absolute/relative paths).
- **`pwd`**: Print the current working directory.
- **`echo`**: Print text to standard output.
- **`type`**: Identify if a command is a builtin or an external executable (and where it is located).
- **`exit`**: Exit the shell with status code 0.

### External Commands
- Executes any binary found in the system `$PATH`.
- Supports execution of local binaries (e.g., `./script.sh`).
- Smart path resolution that handles absolute, relative, and environment-based paths.

### Advanced Parsing & Quoting
- **Single Quotes (`'`)**: Preserves literal values (e.g., `'file with spaces'`).
- **Double Quotes (`"`)**: Handles strings with spaces while allowing for escaped characters.
- **Complex Arguments**: Correctly handles nested quotes and escaped spaces (e.g., `"exe with \'single quotes\'"`).

###  I/O Redirection
Supports standard file redirection operators:
- `>` : Redirect standard output (overwrite).
- `>>`: Redirect standard output (append).
- `<` : Redirect standard input.
- `2>`: Redirect standard error.

---

## Installation & Usage

### Prerequisites
* [Go](https://go.dev/dl/) (1.18 or higher)

### Build and Run
1. **Clone the repository:**
   ```bash
   git clone [repository](https://github.com/denisha-madhura/Gosh.git)
   ```
   cd gosh

Install Dependencies: This project uses google/shlex for POSIX-compliant argument parsing.

```bash
go get [github.com/google/shlex](https://github.com/google/shlex)
```
Build the Shell:

```bash
go build -o Gosh
```
Run:
```bash
./Gosh
```
## Roadmap
We are actively working on making GoShell fully interactive. The following features are currently in development:

- [ ] Pipelines (|): Chaining commands together (e.g., cat file.txt | grep "search").
- [ ] Autocompletion: Tab-completion for commands and file paths.
- [ ] History Persistence:
* Arrow key navigation (Up/Down) to cycle through previous commands.
* Saving command history to a file (~/.gosh_history) between sessions.
- [ ] Environment Variable Expansion: Support for $USER, $HOME, etc.
