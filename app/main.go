package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// 1. Define the necessary types for your new methods
type RedirectionType int

const (
	TokenRedirectOut    RedirectionType = iota // >
	TokenRedirectAppend                        // >>
	TokenRedirectIn                            // <
	TokenRedirect2                             // 2>
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
	"exit": true, "echo": true, "type": true, "cd": true,
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

		// Simple Parser: In a real shell, you'd handle quotes and redirects here
		// For now, let's just split by spaces to create a Command
		parts := strings.Fields(input)
		cmd := &Command{
			Args: parts,
		}

		// Handle 'exit' immediately or via Execute
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
		arg := c.Args[1]
		if Builtins[arg] {
			fmt.Printf("%s is a shell builtin\n", arg)
		} else if path := findInPath(arg); path != "" {
			fmt.Printf("%s is %s\n", arg, path)
		} else {
			fmt.Printf("%s: not found\n", arg)
		}
		return 0
	}
	return 1
}

func (c *Command) executeExternal() int {

	cmdName := c.Args[0]

	// Find command in PATH

	cmdPath := findInPath(cmdName)

	if cmdPath == "" {

		fmt.Fprintf(os.Stdout, "%s: command not found\n", cmdName)

		return 127

	}

	// Create command

	cmd := exec.Command(cmdPath, c.Args[1:]...)

	cmd.Args = c.Args // Set argv to use original command name as argv[0]

	cmd.Stdin = os.Stdin

	cmd.Stdout = os.Stdout

	cmd.Stderr = os.Stderr

	// Apply redirections

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

	// If it's an absolute or relative path, check if it exists

	if strings.Contains(cmdName, "/") || strings.Contains(cmdName, "\\") {

		if isExecutable(cmdName) {

			return cmdName

		}

		return ""

	}

	// Search in PATH

	pathEnv := os.Getenv("PATH")

	paths := filepath.SplitList(pathEnv)

	for _, dir := range paths {

		fullPath := filepath.Join(dir, cmdName)
		if isExecutable(fullPath) {

			return fullPath

		}

		if fullPath+".exe" != fullPath {

			if isExecutable(fullPath + ".exe") {

				return fullPath + ".exe"

			}

		}

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
