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
		if err := execInput(input); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
	}
}

// ErrNoPath is returned when 'cd' was called without a second argument.
var ErrNoPath = errors.New("path required")

var ErrMultipleRedirection = errors.New("multiple redirection of standard output")
var ErrInvalidCommand = errors.New("invalid command")

var redirectInput = ""
var redirectOutput = ""

func writeToDirectory(source, target string) {
	// https://stackoverflow.com/questions/56075774/golang-os-renamefromdir-todir-not-working-in-windows
	origFile, _ := os.ReadFile(source)
	newFile, _ := os.Create(target + "/" + source)
	fmt.Fprintf(newFile, "%s", string(origFile))
}

func redirectIO(source, target string) error {
	// https://stackoverflow.com/questions/56075774/golang-os-renamefromdir-todir-not-working-in-windows
	origFile, _ := os.ReadFile(source)
	newFile, _ := os.Create(target + "/" + source)

	// https://freshman.tech/snippets/go/check-if-file-is-dir/
	_, err := os.Stat(source)
	if err != nil {
		return ErrNoPath
	}

	if target == "" {
		fmt.Printf("%s\n", string(origFile))
		return nil
	}

	_, err = os.Stat(target)
	if err != nil {
		os.Create(target)
	} else {
		os.Remove(target)
	}
	fmt.Fprintf(newFile, "%s", string(origFile))
	return nil
}

func checkRedirection(args []string) error {
	redirectInputIndex := 0  // index of <
	redirectOutputIndex := 0 // index of >
	redirectInputSeen := false
	redirectOutputSeen := false

	// check if there is more than one of the same redirection symbol
	for i, e := range args {
		if e == "<" {
			if !redirectInputSeen {
				redirectInputSeen = true
				redirectInputIndex = i
			} else {
				return ErrMultipleRedirection
			}
		} else if e == ">" {
			if !redirectOutputSeen {
				redirectOutputSeen = true
				redirectOutputIndex = i
			} else {
				return ErrMultipleRedirection
			}
		}
	}

	if redirectInputSeen {
		if redirectInputIndex+1 >= len(args) || redirectInputIndex-1 < 0 {
			return ErrInvalidCommand
		}
		redirectInput = args[redirectInputIndex+1] // follows <
	} else {
		return nil
	}

	if redirectOutputSeen {
		if redirectOutputIndex+1 >= len(args) || redirectOutputIndex-1 < 0 {
			return ErrInvalidCommand
		}
		redirectOutput = args[redirectOutputIndex+1] // follows >
	} else {
		return nil
	}

	return nil
}

func checkAnd(err error, lastArgs int, args []string) error {
	if err != nil {
		return err
	}
	if lastArgs < len(args)-1 {
		if args[lastArgs+1] == "&&" {
			input2 := ""
			for i := lastArgs + 2; i < len(args); i++ {
				input2 = input2 + args[i] + " "
			}
			return execInput(input2)
		}
	}
	return nil
}

func execInput(input string) error {
	// Remove the newline character.
	input = strings.TrimSuffix(input, "\n")
	input = strings.TrimSuffix(input, " ")

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
		return checkAnd(os.Chdir(args[1]), 1, args)
	case "pwd":
		// Get working directory and error
		wd, err := os.Getwd()
		fmt.Println(wd)
		return checkAnd(err, 0, args)
	case "mkdir":
		if len(args) < 2 {
			return ErrNoPath
		}
		// Make the directory and return the error.
		return checkAnd(os.Mkdir(args[1], os.ModePerm), 1, args) // double check ModePerm
	case "mv": //look into not being a path
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
		return checkAnd(nil, 2, args)
	case "rename":
		if len(args) < 3 {
			return ErrNoPath
		}
		return checkAnd(os.Rename(args[1], args[2]), 2, args)
	case "rm":
		if len(args) < 2 {
			return ErrNoPath
		}
		return checkAnd(os.Remove(args[1]), 1, args)
	case "getpid":
		fmt.Println(os.Getpid())
		return checkAnd(nil, 0, args)
	case "setenv":
		if len(args) < 2 {
			return ErrNoPath
		}
		// https://unix.stackexchange.com/questions/368944/what-is-the-difference-between-env-setenv-export-and-when-to-use
		// https://www.geeksforgeeks.org/how-to-split-a-string-in-golang/
		pair := strings.Split(args[1], "=")
		return checkAnd(os.Setenv(pair[0], pair[1]), 1, args)
	case "getenv":
		if len(args) < 2 {
			return ErrNoPath
		}
		value := os.Getenv(args[1])
		if value == "" {
			return nil
		}
		fmt.Println(value)
		return checkAnd(nil, 1, args)
	case "unset":
		if len(args) < 2 {
			return ErrNoPath
		}
		return checkAnd(os.Unsetenv(args[1]), 1, args)
	case "echo":
		if len(args) < 2 {
			fmt.Println()
			return nil
		}
		split := strings.SplitN(input, "\"", 3)
		fmt.Println(split[1])
		return checkAnd(nil, 0, strings.Split(split[2], " "))
	case "ls":
		// https://stackoverflow.com/questions/14668850/list-directory-in-go
		entries, err := os.ReadDir("./")
		if err != nil {
			return err
		}
		for _, e := range entries {
			fmt.Println(e.Name())
		}
		return checkAnd(nil, 0, args)
	case "cat":
		// https://gist.github.com/tetsuok/2795749#file-cat-go-L53
		if len(args) < 2 {
			return ErrNoPath
		}
		err := checkRedirection(args)
		if err != nil {
			return err
		}
		if redirectInput != "" && redirectOutput == "" {
			return redirectIO(redirectInput, redirectOutput)
		}
		if redirectInput != "" && redirectOutput != "" {
			return redirectIO(redirectInput, redirectOutput)
		}
		if redirectInput == "" && redirectOutput != "" {
			return nil
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
		return checkAnd(nil, 1, args)
	case "kill":
		process, err := os.FindProcess(os.Getpid())
		if err != nil {
			return err
		}
		return checkAnd(process.Kill(), 0, args)
	case "exit":
		os.Exit(0)
		return checkAnd(nil, 0, args)
	}

	fmt.Println("hello")
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	return cmd.Run()
}
