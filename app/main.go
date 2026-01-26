package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/chzyer/readline"
)

type bellListener struct {
	completer *readline.PrefixCompleter
	lastTab   string
	tabCount  int
}

func (b *bellListener) OnChange(line []rune, pos int, key rune) (newLine []rune, newPos int, ok bool) {
	if key == 9 {
		lineStr := string(line[:pos])
		if lineStr != b.lastTab {
			b.tabCount = 1
			b.lastTab = lineStr
		} else {
			b.tabCount++
		}

		choices, _ := b.completer.Do(line, pos)
		if len(choices) == 0 {
			fmt.Print("\x07")
			return line, pos, true
		}

		if len(choices) > 1 {
			if b.tabCount == 1 {
				fmt.Print("\x07")
				return line, pos, true
			} else {
				results := make([]string, 0, len(choices))
				for _, c := range choices {
					suggestion := strings.TrimSpace(string(c))
					results = append(results, lineStr+suggestion)
				}
				sort.Strings(results)
				fmt.Printf("\n%s\n$ %s", strings.Join(results, "  "), lineStr)
				return line, pos, true
			}
		}
	} else {
		b.tabCount = 0
		b.lastTab = ""
	}
	return nil, 0, false
}

type RedirectionType int

const (
	TokenRedirectOut RedirectionType = iota
	TokenRedirectAppend
	TokenRedirectIn
	TokenRedirect2
	TokenRedirect22
)

type Redirection struct {
	Type     RedirectionType
	Filename string
}

type Command struct {
	Args         []string
	Redirections []Redirection
}

var Builtins = map[string]bool{
	"exit": true, "echo": true, "type": true, "cd": true, "pwd": true, "history": true,
}

var historyEntries []string
var sessionStartIndex int

func getPathExecutables() []readline.PrefixCompleterInterface {
	var items []readline.PrefixCompleterInterface
	for name := range Builtins {
		items = append(items, readline.PcItem(name))
	}
	pathEnv := os.Getenv("PATH")
	paths := filepath.SplitList(pathEnv)
	seen := make(map[string]bool)
	for _, dir := range paths {
		files, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, f := range files {
			if !f.IsDir() && !seen[f.Name()] {
				items = append(items, readline.PcItem(f.Name()))
				seen[f.Name()] = true
			}
		}
	}
	return items
}

func InputParser(input string) []string {
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
	return newArr
}

func parsePipeline(input string) [][]string {
	var parts [][]string
	var currentPart []string
	rawTokens := InputParser(input)
	for _, token := range rawTokens {
		if token == "|" {
			parts = append(parts, currentPart)
			currentPart = []string{}
		} else {
			currentPart = append(currentPart, token)
		}
	}
	parts = append(parts, currentPart)
	return parts
}

func loadHistory(path string) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if line != "" {
			historyEntries = append(historyEntries, line)
		}
	}
	sessionStartIndex = len(historyEntries)
}

func saveHistory(path string, appendOnly bool) {
	flags := os.O_CREATE | os.O_WRONLY
	startIdx := 0
	if appendOnly {
		flags |= os.O_APPEND
		startIdx = sessionStartIndex
	} else {
		flags |= os.O_TRUNC
	}
	file, err := os.OpenFile(path, flags, 0644)
	if err != nil {
		return
	}
	defer file.Close()

	for i := startIdx; i < len(historyEntries); i++ {
		fmt.Fprintln(file, historyEntries[i])
	}

	if appendOnly {
		sessionStartIndex = len(historyEntries)
	}
}

func main() {
	histFile := os.Getenv("HISTFILE")
	if histFile != "" {
		loadHistory(histFile)
	} else {
		sessionStartIndex = 0
	}

	completer := readline.NewPrefixCompleter(getPathExecutables()...)
	l := &bellListener{completer: completer}
	rl, err := readline.NewEx(&readline.Config{
		Prompt:       "$ ",
		AutoComplete: completer,
		Listener:     l,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer rl.Close()

	for {
		input, err := rl.Readline()
		if err != nil {
			if histFile != "" {
				saveHistory(histFile, false)
			}
			break
		}
		rawInput := input
		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}

		historyEntries = append(historyEntries, rawInput)

		if input == "exit" {
			if histFile != "" {
				saveHistory(histFile, false)
			}
			os.Exit(0)
		}

		pipelineParts := parsePipeline(input)
		executePipeline(pipelineParts)
	}
}

func executePipeline(parts [][]string) {
	var nextInput io.ReadCloser
	var wg sync.WaitGroup

	for i, part := range parts {
		isLast := i == len(parts)-1
		var r *io.PipeReader
		var w *io.PipeWriter
		if !isLast {
			r, w = io.Pipe()
		}
		cmdObj := buildCommand(part)
		if len(cmdObj.Args) == 0 {
			continue
		}

		wg.Add(1)
		if !isLast {
			go func(c Command, in io.ReadCloser, out io.WriteCloser) {
				defer wg.Done()
				c.Execute(in, out, os.Stderr)
				out.Close()
				if in != nil {
					in.Close()
				}
			}(cmdObj, nextInput, w)
			nextInput = r
		} else {
			c := cmdObj
			in := nextInput
			c.Execute(in, os.Stdout, os.Stderr)
			if in != nil {
				in.Close()
			}
			wg.Done()
		}
	}
	wg.Wait()
}

func buildCommand(parts []string) Command {
	cmd := Command{Args: []string{}, Redirections: []Redirection{}}
	for i := 0; i < len(parts); i++ {
		word := parts[i]
		if (word == ">" || word == "1>" || word == ">>" || word == "1>>" || word == "2>" || word == "2>>" || word == "<") && i+1 < len(parts) {
			var rType RedirectionType
			switch word {
			case ">", "1>":
				rType = TokenRedirectOut
			case ">>", "1>>":
				rType = TokenRedirectAppend
			case "2>":
				rType = TokenRedirect2
			case "2>>":
				rType = TokenRedirect22
			case "<":
				rType = TokenRedirectIn
			}
			cmd.Redirections = append(cmd.Redirections, Redirection{Type: rType, Filename: parts[i+1]})
			i++
		} else {
			cmd.Args = append(cmd.Args, word)
		}
	}
	return cmd
}

func (c *Command) Execute(stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	if len(c.Args) == 0 {
		return 0
	}
	if Builtins[c.Args[0]] {
		return c.executeBuiltin(stdin, stdout, stderr)
	}
	return c.executeExternal(stdin, stdout, stderr)
}

func (c *Command) executeBuiltin(stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	for _, redir := range c.Redirections {
		os.MkdirAll(filepath.Dir(redir.Filename), 0755)
		switch redir.Type {
		case TokenRedirectOut:
			f, _ := os.Create(redir.Filename)
			defer f.Close()
			stdout = f
		case TokenRedirectAppend:
			f, _ := os.OpenFile(redir.Filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			defer f.Close()
			stdout = f
		case TokenRedirect2:
			f, _ := os.Create(redir.Filename)
			defer f.Close()
			stderr = f
		case TokenRedirect22:
			f, _ := os.OpenFile(redir.Filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			defer f.Close()
			stderr = f
		}
	}
	switch c.Args[0] {
	case "history":
		if len(c.Args) > 2 {
			flag := c.Args[1]
			path := c.Args[2]
			switch flag {
			case "-r":
				loadHistory(path)
			case "-w":
				saveHistory(path, false)
			case "-a":
				saveHistory(path, true)
			}
			return 0
		}
		n := len(historyEntries)
		if len(c.Args) > 1 {
			if val, err := strconv.Atoi(c.Args[1]); err == nil {
				n = val
			}
		}
		start := 0
		if n < len(historyEntries) {
			start = len(historyEntries) - n
		}
		for i := start; i < len(historyEntries); i++ {
			fmt.Fprintf(stdout, "%5d  %s\n", i+1, historyEntries[i])
		}
	case "echo":
		fmt.Fprintln(stdout, strings.Join(c.Args[1:], " "))
	case "type":
		if len(c.Args) < 2 {
			return 0
		}
		arg := c.Args[1]
		if Builtins[arg] {
			fmt.Fprintf(stdout, "%s is a shell builtin\n", arg)
		} else if path := findInPath(arg); path != "" {
			fmt.Fprintf(stdout, "%s is %s\n", arg, path)
		} else {
			fmt.Fprintf(stdout, "%s: not found\n", arg)
		}
	case "pwd":
		output, _ := os.Getwd()
		fmt.Fprintf(stdout, "%s\n", output)
	case "cd":
		path := ""
		if len(c.Args) == 1 || c.Args[1] == "~" {
			path = os.Getenv("HOME")
		} else {
			path = c.Args[1]
		}
		if err := os.Chdir(path); err != nil {
			fmt.Fprintf(stderr, "cd: %s: No such file or directory\n", path)
			return 1
		}
	}
	return 0
}

func (c *Command) executeExternal(stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	cmdPath := findInPath(c.Args[0])
	if cmdPath == "" {
		fmt.Fprintf(stderr, "%s: command not found\n", c.Args[0])
		return 127
	}
	cmd := exec.Command(cmdPath, c.Args[1:]...)
	cmd.Args[0] = c.Args[0]
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	for _, redir := range c.Redirections {
		os.MkdirAll(filepath.Dir(redir.Filename), 0755)
		switch redir.Type {
		case TokenRedirectOut:
			f, _ := os.Create(redir.Filename)
			defer f.Close()
			cmd.Stdout = f
		case TokenRedirectAppend:
			f, _ := os.OpenFile(redir.Filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			defer f.Close()
			cmd.Stdout = f
		case TokenRedirectIn:
			f, _ := os.Open(redir.Filename)
			defer f.Close()
			cmd.Stdin = f
		case TokenRedirect2:
			f, _ := os.Create(redir.Filename)
			defer f.Close()
			cmd.Stderr = f
		case TokenRedirect22:
			f, _ := os.OpenFile(redir.Filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			defer f.Close()
			cmd.Stderr = f
		}
	}
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode()
		}
		return 1
	}
	return 0
}

func findInPath(cmdName string) string {
	if filepath.IsAbs(cmdName) || strings.HasPrefix(cmdName, "./") {
		if _, err := os.Stat(cmdName); err == nil {
			return cmdName
		}
	}
	pathEnv := os.Getenv("PATH")
	for _, dir := range filepath.SplitList(pathEnv) {
		fullPath := filepath.Join(dir, cmdName)
		info, err := os.Stat(fullPath)
		if err == nil && !info.IsDir() && info.Mode().Perm()&0111 != 0 {
			return fullPath
		}
	}
	return ""
}
