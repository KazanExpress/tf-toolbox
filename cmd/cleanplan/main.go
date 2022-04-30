package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

func main() {
	info, err := os.Stdin.Stat()
	if err != nil {
		panic(err)
	}

	if info.Mode()&os.ModeCharDevice != 0 {
		fmt.Println("The command is intended to work with pipes.")
		fmt.Println("Usage: cat plan.txt | cleanplan")
		return
	}

	reader := bufio.NewReader(os.Stdin)
	var output []rune

	for {
		input, _, err := reader.ReadRune()
		if err != nil && err == io.EOF {
			break
		}
		output = append(output, input)
	}

	str := string(output)
	lines := strings.Split(str, "\n")

	for j := 0; j < len(lines); j++ {
		if j > 2 && lines[j] == lines[j-1] && lines[j-2] == lines[j-1] {
			continue
		}
		fmt.Println(lines[j])
	}
}
