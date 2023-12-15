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

// ErrNoPath is returned when 'cd' was called without a second argument.
var ErrNoPath = errors.New("path required")
var ErrMultipleRedirection = errors.New("multiple redirection of standard output")
var ErrInvalidCommand = errors.New("invalid command")

func main() {
	fmt.Print("> ")
	reader := bufio.NewReader(os.Stdin)
	for {
		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
		input = strings.TrimSuffix(input, "\n")
		input = strings.TrimSuffix(input, " ")
		args := strings.Split(input, " ")
		if args[len(args)-1] == "&" {
			input2 := ""
			for i := 0; i < len(args)-1; i++ {
				input2 = input2 + args[i] + " "
			}
			go PrintError(input)
			fmt.Print("> ")
		} else { // Handle the execution of the input.
			PrintError(input)
		}
	}
}

func PrintError(input string) {
	if err := execInput(input); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	fmt.Print("> ")
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

func handleOutput(output []string, lastArgs int, args []string) error {
	if lastArgs < len(args)-1 {
		first := true
		echo := false
		if args[lastArgs+1] == "|" {
			input2 := ""
			for i := lastArgs + 2; i < len(args); i++ {
				if first {
					input2 = input2 + args[i] + " "
					if args[i] == "echo" {
						echo = true
						input2 = input2 + "\""
					}
					for i := 0; i < len(output); i++ {
						input2 = input2 + output[i] + " "
					}
					if echo {
						input2 = input2 + "\""
					}
				} else {
					input2 = input2 + args[i] + " "
				}
			}
			return execInput(input2)
		}
		redirectInput, redirectOutput, err := checkRedirection(args)
		if redirectInput == "" {

		}
		if err != nil {
			return err
		}
		if redirectOutput != "" {
			_, err := os.Stat(redirectOutput)
			if err == nil {
				os.Remove(redirectOutput)
			}
			newFile, err := os.Create(redirectOutput)
			if err != nil {
				return err
			}
			for i := 0; i < len(output); i++ {
				fmt.Fprintf(newFile, output[i])
			}
			return nil
		}
	}
	for i := 0; i < len(output); i++ {
		fmt.Print(output[i])
	}
	return nil
}

func execInput(input string) error {
	// Remove the newline character.
	input = strings.TrimSuffix(input, "\n")
	input = strings.TrimSuffix(input, " ")

	// Split the input separate the command and the arguments.
	args := strings.Split(input, " ")

	args = separateSpecialSigns(args)

	output := make([]string, 1)

	// Check for built-in commands.
	switch args[0] {
	case "cd":
		if len(args) < 2 {
			return os.Chdir("/")
		}
		if args[1] == "&&" {
			return checkAnd(os.Chdir("/"), 0, args)
		}
		// Change the directory and return the error.
		return checkAnd(os.Chdir(args[1]), 1, args)
	case "pwd":
		// Get working directory and error
		wd, err := os.Getwd()
		if err != nil {
			return err
		}
		output[0] = wd + "\n"
		err = handleOutput(output, 0, args)
		if err != nil {
			return err
		}
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
		output[0] = fmt.Sprint(os.Getpid()) + "\n"
		err := handleOutput(output, 0, args)
		if err != nil {
			return err
		}
		return checkAnd(nil, 0, args)
	case "setenv":
		if len(args) < 2 {
			return ErrNoPath
		}
		// https://unix.stackexchange.com/questions/368944/what-is-the-difference-between-env-setenv-export-and-when-to-use
		// https://www.geeksforgeeks.org/how-to-split-a-string-in-golang/
		pair, err := getKeyValue(args)
		if err != nil {
			return err
		}
		return checkAnd(os.Setenv(pair[0], pair[1]), 1, args)
	case "getenv":
		if len(args) < 2 {
			return ErrNoPath
		}
		value := os.Getenv(args[1])
		if value == "" {
			return nil
		}
		output[0] = value + "\n"
		err := handleOutput(output, 1, args)
		if err != nil {
			return err
		}
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
		output[0] = split[1] + "\n"
		err := handleOutput(output, 0, strings.Split(split[2], " "))
		if err != nil {
			return err
		}
		return checkAnd(nil, 0, strings.Split(split[2], " "))
	case "ls":
		// https://stackoverflow.com/questions/14668850/list-directory-in-go
		entries, err := os.ReadDir("./")
		if err != nil {
			return err
		}
		first := true
		for _, e := range entries {
			if first {
				output[0] = e.Name() + "\n"
				first = false
			} else {
				output = append(output, e.Name()+"\n")
			}
		}
		err = handleOutput(output, 0, args)
		if err != nil {
			return err
		}
		return checkAnd(nil, 0, args)
	case "cat":
		// not working
		// https://gist.github.com/tetsuok/2795749#file-cat-go-L53
		if len(args) < 2 {
			return ErrNoPath
		}
		for i := 1; i < len(args); i++ {
			if args[i] == "|" {
				break
			}
			if args[i] == "<" {
				break
			}
			if args[i] == ">" {
				break
			}
			file, err := os.Open(args[i])
			if err != nil {
				return err
			}
			defer file.Close()
			rd := bufio.NewReader(file)
			first := true
			for {
				line, err := rd.ReadString('\n')
				if err == io.EOF {
					if first {
						output[0] = line + "\n"
					} else {
						output = append(output, line+"\n")
					}
					break
				}
				if err != nil {
					return err
				}
				if first {
					output[0] = line
					first = false
				} else {
					output = append(output, line)
				}
			}
		}
		err := handleOutput(output, 1, args)
		if err != nil {
			return err
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

func getKeyValue(args []string) ([]string, error) {
	keyval := []string{}
	equalsSeen := false
	for i, e := range args {
		if e == "=" {
			equalsSeen = true
			if i-1 <= 0 {
				return keyval, ErrInvalidCommand
			}
			if i+1 >= len(args) {
				return keyval, ErrInvalidCommand
			}
			keyval = append(keyval, args[i-1])
			keyval = append(keyval, args[i+1])
			break
		}
	}
	if !equalsSeen {
		return keyval, ErrInvalidCommand
	}
	return keyval, nil
}

func writeToDirectory(source, target string) {
	// https://stackoverflow.com/questions/56075774/golang-os-renamefromdir-todir-not-working-in-windows
	origFile, _ := os.ReadFile(source)
	newFile, _ := os.Create(target + "/" + source)
	fmt.Fprintf(newFile, "%s", string(origFile))
}

func separateSpecialSigns(args []string) []string {
	newArgs := []string{}
	tmpArgs := []string{}
	for _, e := range args {
		temp := 0
		if string(e[0]) == "\"" {
			inString := true
			for i, c := range e {
				if i == 0 {
					continue
				}
				if string(c) == "\"" {
					inString = !inString
					if inString && i != 0 {
						tmpArgs = append(tmpArgs, e[temp+1:i])
					} else if !inString {
						tmpArgs = append(tmpArgs, e[temp:i+1])
					}
					temp = i
					continue
				}
			}
		} else {
			tmpArgs = append(tmpArgs, e)
		}
	}
	for _, e := range tmpArgs {
		if string(e[0]) == "\"" || e == "<" || e == ">" || e == "|" || e == "=" {
			newArgs = append(newArgs, e)
			continue
		}
		specialSignSeen := false
		for i := 0; i < len(e); i++ {
			if string(e[i]) == "<" || string(e[i]) == ">" || string(e[i]) == "|" || string(e[i]) == "=" {
				specialSignSeen = true
				// https://stackoverflow.com/questions/55212090/string-splitting-before-character
				// https://www.tutorialspoint.com/how-to-trim-a-string-in-golang#:~:text=Using%20the%20strings.,trailing%20whitespace%20from%20a%20string.
				if i != 0 {
					newArgs = append(newArgs, e[:i])
				}
				newArgs = append(newArgs, string(e[i]))
				if i+1 < len(e) {
					newArgs = append(newArgs, e[i+1:])
				}
			}
		}
		if !specialSignSeen {
			newArgs = append(newArgs, e)
		}
	}
	return newArgs
}

func checkRedirection(args []string) (string, string, error) {
	redirectInput := ""
	redirectOutput := ""
	redirectInputSignIndex := -1
	redirectOutputSignIndex := -1
	redirectInputSeen := false
	redirectOutputSeen := false

	// check if there is more than one of the same redirection symbol
	for i, e := range args {
		if e == "<" {
			if !redirectInputSeen {
				redirectInputSeen = true
				redirectInputSignIndex = i
			} else {
				return redirectInput, redirectOutput, ErrMultipleRedirection
			}
		} else if e == ">" {
			if !redirectOutputSeen {
				redirectOutputSeen = true
				redirectOutputSignIndex = i
			} else {
				return redirectInput, redirectOutput, ErrMultipleRedirection
			}
		}
	}

	if redirectInputSeen {
		if redirectInputSignIndex+1 >= len(args) || redirectInputSignIndex-1 < 0 {
			return redirectInput, redirectOutput, ErrInvalidCommand
		}
		redirectInput = args[redirectInputSignIndex+1]
	}
	if redirectOutputSeen {
		if redirectOutputSignIndex+1 >= len(args) || redirectOutputSignIndex-1 < 0 {
			return redirectInput, redirectOutput, ErrInvalidCommand
		}
		redirectOutput = args[redirectOutputSignIndex+1]
	}
	if !redirectInputSeen && redirectOutputSeen {
		if redirectOutputSignIndex-1 < 0 {
			return redirectInput, redirectOutput, ErrInvalidCommand
		}
		if redirectOutputSignIndex-1 > 0 {
			redirectInput = args[redirectOutputSignIndex-1]
		} else if redirectOutputSignIndex+2 < len(args) {
			redirectInput = args[redirectOutputSignIndex+2]
		}
	}
	return redirectInput, redirectOutput, nil
}
