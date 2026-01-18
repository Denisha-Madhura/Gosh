package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// Ensures gofmt doesn't remove the "fmt" import in stage 1 (feel free to remove this!)
var _ = fmt.Print

func main() {
	// TODO: Uncomment the code below to pass the first stage
	for {
		fmt.Print("$ ")

		command, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error reading input: ", err)
			os.Exit(1)
		}
		command = strings.TrimSpace(command)
		switch {
		case command == "exit":
			os.Exit(0)

		case strings.HasPrefix(command, "echo"):
			fmt.Println(strings.TrimPrefix(command, "echo "))
		case strings.HasPrefix(command, "type"):
			if strings.HasSuffix(command, "exit") || strings.HasSuffix(command, "echo") || strings.HasSuffix(command, "type") {
				fmt.Println(strings.TrimPrefix(command, "type ") + " is a shell builtin")
			} else {
				fmt.Println(strings.TrimPrefix(command, "type ") + ": not found")
			}
		default:
			fmt.Println(command + ": command not found")
		}
	}
}
