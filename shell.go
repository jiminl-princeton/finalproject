// https://blog.init-io.net/post/2018/07-01-go-unix-shell/

// hi

package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func main() {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("> ")
		// Read the keyboad input.
		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		}

		// Handle the execution of the input.
		if err = execInput(input); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
	}
}

// ErrNoPath is returned when 'cd' was called without a second argument.
var ErrNoPath = errors.New("path required")

// var ErrPathNotFound = errors.New("path not found")

func execInput(input string) error {
	// Remove the newline character.
	input = strings.TrimSuffix(input, "\n")

	// Split the input separate the command and the arguments.
	args := strings.Split(input, " ")

	// Check for built-in commands.
	switch args[0] {
	case "cd":
		// 'cd' to home with empty path not yet supported.
		if len(args) < 2 {
			return ErrNoPath
		}
		// Change the directory and return the error.
		return os.Chdir(args[1])
	case "exit":
		os.Exit(0)
	}

	// for i, e := range args {
	// 	if e == "<" {
	// 		if i+1 >= len(args) || i-1 < 0 {
	// 			return ErrNoPath
	// 		}
	// 		// https://freshman.tech/snippets/go/check-if-file-is-dir/
	// 		_, err := os.Stat(args[i+1])
	// 		if err != nil {
	// 			cmd := exec.Command(fmt.Sprintf("touch %s", args[i+1]))
	// 			cmd.Run()
	// 		}
	// 		_, err = os.Stat(args[i-1])
	// 		if err != nil {
	// 			return ErrPathNotFound
	// 		}
	// 		// https://stackoverflow.com/questions/38288012/exec-command-with-input-redirection
	// 		bytes, err := os.ReadFile(args[i+1])
	// 		if err != nil {
	// 			log.Fatal(err)
	// 		}
	// 		cmd := exec.Command(args[i-1])
	// 		stdin, err := cmd.StdinPipe()
	// 		if err != nil {
	// 			log.Fatal(err)
	// 		}
	// 		err = cmd.Start()
	// 		if err != nil {
	// 			log.Fatal(err)
	// 		}
	// 		_, err = io.WriteString(stdin, string(bytes))
	// 		if err != nil {
	// 			log.Fatal(err)
	// 		}
	// 	}
	// }

	// Prepare the command to execute.
	cmd := exec.Command(args[0], args[1:]...)

	// Set the correct output device.
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	// Execute the command and return the error.
	return cmd.Run()
}
