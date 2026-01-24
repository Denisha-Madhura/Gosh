package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/google/shlex"
)

type RedirectionType int

const (
	TokenRedirectOut RedirectionType = iota
	TokenRedirectAppend
	TokenRedirectIn
	TokenRedirect2
)

type Redirection struct {
	Type     RedirectionType
	Filename string
}

type Command struct {
	Args         []string
	Redirections []Redirection
}

type Shell struct {
}

var Builtins = map[string]bool{
	"exit": true, "echo": true, "type": true, "cd": true, "pwd": true,
}

func InputParser(input string) (string, []string) {
	var word strings.Builder
	var newArr []string
	preserveNextLiteral := false
	backslashInQuotes := false
	inQuotes := false
	quoteChar := rune(0)

	for _, ch := range input {

		if preserveNextLiteral {
			word.WriteRune(ch)
			preserveNextLiteral = false
			continue
		}
		if backslashInQuotes {
			if ch == '$' || ch == '\\' || ch == '"' || ch == '`' {
				word.WriteRune(ch)
			} else {
				word.WriteRune('\\')
				word.WriteRune(ch)
			}
			backslashInQuotes = false
			continue
		}

		switch {
		case ch == '"' || ch == '\'':
			if !inQuotes {
				inQuotes = true
				quoteChar = ch
			} else if ch == quoteChar {
				inQuotes = false
				quoteChar = rune(0)
			} else {
				word.WriteRune(ch)
			}
		case ch == '\\':
			if !inQuotes {
				preserveNextLiteral = true
			} else if quoteChar == '"' {
				backslashInQuotes = true
			} else {
				word.WriteRune(ch)
			}
		case ch == ' ':
			if inQuotes {
				word.WriteRune(ch)
			} else if word.Len() > 0 {
				newArr = append(newArr, word.String())
				word.Reset()
			}
		default:
			word.WriteRune(ch)
		}
	}

	if word.Len() > 0 {
		newArr = append(newArr, word.String())
	}
	if len(newArr) == 0 {
		return "", nil
	}

	noSingles := strings.ReplaceAll(input, "'", "")
	noDoubles := strings.ReplaceAll(noSingles, `"`, "")
	output := noDoubles

	return output, newArr
}


func main() {
	shell := &Shell{}
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("$ ")
		input, err := reader.ReadString('\n')
		if err != nil {
			break
		}

		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}

		cmdArr := strings.Fields(input)
		cmd := strings.TrimSpace(cmdArr[0])

		var args []string

		newInput, newArgsArr := InputParser(input[:len(input)-1])

		if strings.Contains(input, "'") || strings.Contains(input, `"`) || strings.Contains(input, `/`) || strings.Contains(input, `\`) {
			input = newInput
			args = newArgsArr[1:]
			cmd = newArgsArr[0]
		} else if len(newArgsArr) == 0 {
			args = cmdArr[1:]
		} else {
			args = cmdArr[1:]
		}

		if strings.TrimSpace(input) == "exit 0" {
			break
		}
		}

		if len(cmd.Args) == 0 {
			continue
		}

		if cmd.Args[0] == "exit" {
			os.Exit(0)
		}

		cmd.Execute(shell)
	}
}

func (c *Command) Execute(shell *Shell) int {
	if len(c.Args) == 0 {
		return 0
	}

	cmdName := c.Args[0]

	if Builtins[cmdName] {
		return c.executeBuiltin(shell)
	}

	return c.executeExternal()
}

func (c *Command) executeBuiltin(shell *Shell) int {
	cmdName := c.Args[0]
	switch cmdName {
	case "echo":
		fmt.Println(strings.Join(c.Args[1:], " "))
		return 0
	case "type":
		if len(c.Args) < 2 {
			return 0
		}
		arg := c.Args[1]
		if Builtins[arg] {
			fmt.Printf("%s is a shell builtin\n", arg)
		} else if path := findInPath(arg); path != "" {
			fmt.Printf("%s is %s\n", arg, path)
		} else {
			fmt.Printf("%s: not found\n", arg)
		}
		return 0
	case "pwd":
		output, _ := os.Getwd()
		fmt.Printf("%s\n", output)
		return 0
	case "cd":
		return doCd(c.Args[1:])
	}
	return 1
}

func (c *Command) executeExternal() int {
	cmdName := c.Args[0]
	cmdPath := findInPath(cmdName)

	if cmdPath == "" {
		fmt.Fprintf(os.Stdout, "%s: command not found\n", cmdName)
		return 127
	}

	cmd := exec.Command(cmdPath, c.Args[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	for _, redir := range c.Redirections {
		switch redir.Type {
		case TokenRedirectOut:
			f, err := os.Create(redir.Filename)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s: %s: %s\n", cmdName, redir.Filename, err)
				return 1
			}
			defer f.Close()
			cmd.Stdout = f
		case TokenRedirectAppend:
			f, err := os.OpenFile(redir.Filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s: %s: %s\n", cmdName, redir.Filename, err)
				return 1
			}
			defer f.Close()
			cmd.Stdout = f
		case TokenRedirectIn:
			f, err := os.Open(redir.Filename)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s: %s: No such file or directory\n", cmdName, redir.Filename)
				return 1
			}
			defer f.Close()
			cmd.Stdin = f
		case TokenRedirect2:
			f, err := os.Create(redir.Filename)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s: %s: %s\n", cmdName, redir.Filename, err)
				return 1
			}
			defer f.Close()
			cmd.Stderr = f
		}
	}

	err := cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode()
		}
		return 1
	}

	return 0
}

func findInPath(cmdName string) string {
	if filepath.IsAbs(cmdName) || strings.HasPrefix(cmdName, "./") || strings.HasPrefix(cmdName, "../") {
		if isExecutable(cmdName) {
			return cmdName
		}
		return ""
	}

	pathEnv := os.Getenv("PATH")
	paths := filepath.SplitList(pathEnv)

	for _, dir := range paths {
		fullPath := filepath.Join(dir, cmdName)
		if isExecutable(fullPath) {
			return fullPath
		}
	}

	if isExecutable(cmdName) {
		return cmdName
	}

	return ""
}

func isExecutable(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	if !info.Mode().IsRegular() {
		return false
	}
	if info.Mode().Perm()&0111 != 0 {
		return true
	}
	return false
}

func doCd(args []string) int {
	path := ""
	if len(args) == 0 {
		path = os.Getenv("HOME")
	} else {
		path = args[0]
	}

	if path == "~" {
		path = os.Getenv("HOME")
	}

	if err := os.Chdir(path); err != nil {
		fmt.Fprintf(os.Stderr, "cd: %s: No such file or directory\n", path)
		return 1
	}
	return 0
}
