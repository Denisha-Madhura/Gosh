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
		switch {
		case command == "exit\n":
			os.Exit(0)

		case strings.HasPrefix(command, "echo"):
			fmt.Print(strings.TrimPrefix(command, "echo "))
		default:
			fmt.Println(command[:len(command)-1] + ": command not found")
		}
	}
}
