// https://blog.init-io.net/post/2018/07-01-go-unix-shell/

package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
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
		if err := execInput2(input); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
	}
}

// ErrNoPath is returned when 'cd' was called without a second argument.
var ErrNoPath = errors.New("path required")

var ErrInvalidCommand = errors.New("invalid command")

var lastArgs = 0

func writeToDirectory(source, target string) {
	// https://stackoverflow.com/questions/56075774/golang-os-renamefromdir-todir-not-working-in-windows
	origFile, _ := os.ReadFile(source)
	newFile, _ := os.Create(target + "/" + source)
	fmt.Fprintf(newFile, "%s", string(origFile))
}

func writeToFile(source, target string) error {
	// https://freshman.tech/snippets/go/check-if-file-is-dir/
	// source path (must exist)
	_, err := os.Stat(source)
	if err != nil {
		return ErrNoPath
	}
	// target path (need not exist)
	_, err = os.Stat(target)
	if err != nil {
		os.Create(target)
	} else {
		os.Remove(target)
	}
	// https://stackoverflow.com/questions/56075774/golang-os-renamefromdir-todir-not-working-in-windows
	origFile, _ := os.ReadFile(source)
	newFile, _ := os.Create(target + "/" + source)
	fmt.Fprintf(newFile, "%s", string(origFile))
	return nil
}

func redirectIO(args []string) error {
	redirectInput := -1  // index of < symbol
	redirectOutput := -1 // index of > symbol
	redirectInputSeen := false
	redirectOutputSeen := false

	// check if there is more than one of the same redirection symbol
	for i, e := range args {
		if e == "<" {
			if !redirectInputSeen {
				redirectInputSeen = true
				redirectInput = i
			} else {
				return ErrInvalidCommand
			}
		} else if e == ">" {
			if !redirectOutputSeen {
				redirectOutputSeen = true
				redirectOutput = i
			} else {
				return ErrInvalidCommand
			}
		}
	}

	if redirectInput != -1 {
		if redirectInput+1 >= len(args) || redirectInput-1 < 0 {
			return ErrNoPath
		}
		source := args[redirectInput+1]
		target := args[redirectInput-1]
		writeToFile(source, target)
	}

	if redirectOutput != -1 {
		if redirectInput+1 >= len(args) || redirectInput-1 < 0 {
			return ErrNoPath
		}
		source := args[redirectInput-1]
		target := args[redirectInput+1]
		writeToFile(source, target)
	}

	return nil
}

func execInput2(input string) error {
	if err := execInput2(input); err != nil {
		return err
	}
	args := strings.Split(input, " ")
	if lastArgs < len(args)-1 {
		if args[lastArgs+1] == "&&" {
			input2 := ""
			for i := lastArgs + 2; i < len(args); i++ {
				input2 = input2 + args[i]
			}
			return execInput2(input2)
		}
	}
	return nil
}

func execInput(input string) error {
	// Remove the newline character.
	input = strings.TrimSuffix(input, "\n")

	// Split the input separate the command and the arguments.
	args := strings.Split(input, " ")

	// Check for built-in commands.
	switch args[0] {
	case "cd":
		lastArgs = 1
		// 'cd' to home with empty path not yet supported.
		if len(args) < 2 {
			return ErrNoPath
		}
		// Change the directory and return the error.
		return os.Chdir(args[1])
	case "pwd":
		lastArgs = 0
		// Get working directory and error
		wd, err := os.Getwd()
		fmt.Println(wd)
		return err
	case "mkdir":
		lastArgs = 1
		if len(args) < 2 {
			return ErrNoPath
		}
		// Make the directory and return the error.
		return os.Mkdir(args[1], os.ModePerm) // double check ModePerm
	case "mv":
		lastArgs = 2
		if len(args) < 3 {
			return ErrNoPath
		}
		// Check if second argument is a file or directory
		// https://www.tutorialspoint.com/golang-program-to-check-if-a-file-is-directory-or-a-file
		info, err := os.Stat(args[2])
		if err != nil {
			return err
		}
		if !info.IsDir() {
			return os.Rename(args[1], args[2])
		}
		writeToDirectory(args[1], args[2])
		os.Remove(args[1])
		return nil
	case "rename":
		lastArgs = 2
		if len(args) < 3 {
			return ErrNoPath
		}
		return os.Rename(args[1], args[2])
	case "rm":
		lastArgs = 1
		if len(args) < 2 {
			return ErrNoPath
		}
		return os.Remove(args[1])
	case "getpid":
		lastArgs = 0
		fmt.Println(os.Getpid())
		return nil
	case "setenv":
		lastArgs = 1
		if len(args) < 2 {
			return ErrNoPath
		}
		// https://unix.stackexchange.com/questions/368944/what-is-the-difference-between-env-setenv-export-and-when-to-use
		// https://www.geeksforgeeks.org/how-to-split-a-string-in-golang/
		pair := strings.Split(args[1], "=")
		return os.Setenv(pair[0], pair[1])
	case "getenv":
		lastArgs = 1
		if len(args) < 2 {
			return ErrNoPath
		}
		value := os.Getenv(args[1])
		if value == "" {
			return nil
		}
		fmt.Println(value)
		return nil
	case "unset":
		lastArgs = 1
		if len(args) < 2 {
			return ErrNoPath
		}
		return os.Unsetenv(args[1])
	case "echo":
		lastArgs = 1
		if len(args) < 2 {
			fmt.Println()
			return nil
		}
		split := strings.Split(input, "\"")
		fmt.Println(split[1])
		return nil
	case "ls":
		lastArgs = 0
		// https://stackoverflow.com/questions/14668850/list-directory-in-go
		entries, err := os.ReadDir("./")
		if err != nil {
			return err
		}
		for _, e := range entries {
			fmt.Println(e.Name())
		}
		return nil
	case "cat":
		// not working
		// https://gist.github.com/tetsuok/2795749#file-cat-go-L53
		lastArgs = 1
		if len(args) < 2 {
			return ErrNoPath
		}
		for i := 1; i < len(args); i++ {
			file, err := os.Open(args[i])
			if err != nil {
				return err
			}
			defer file.Close()
			rd := bufio.NewReader(file)
			for {
				line, err := rd.ReadString('\n')
				if err == io.EOF {
					fmt.Printf("%s\n", line)
					break
				}
				if err != nil {
					return err
				}
				fmt.Printf("%s", line)
			}
		}
		return nil
	case "kill":
		lastArgs = 0
		process, err := os.FindProcess(os.Getpid())
		if err != nil {
			return err
		}
		return process.Kill()
	case "exit":
		lastArgs = 0
		os.Exit(0)
		return nil
	}

	fmt.Println("hello")
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	return cmd.Run()
}
