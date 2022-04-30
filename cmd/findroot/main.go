package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// FindOptimalPath returns optimal path to set TF_ROOT
// Если изменены файлы только в одной из дочерних дирикторий в environments/
// тогда оптимальный путь будет environments/child иначе environments
func FindOptimalPath(envsPath string, changedPaths, envs []string) string {
	const pathSep = "/"

	bucket := map[string]int{}
	last := ""

	envsMap := map[string]bool{}
	for _, env := range envs {
		envsMap[env] = true
	}

	for _, p := range changedPaths {

		subPaths := strings.SplitN(p, pathSep, 3)

	innerloop:
		for _, substr := range subPaths {
			if substr == envsPath {
				continue
			}

			bucket[substr] += 1
			last = substr
			break innerloop
		}
	}

	if len(bucket) == 1 && envsMap[last] {
		return fmt.Sprintf("%s%s%s", envsPath, pathSep, last)
	}
	return fmt.Sprintf("%s", envsPath)
}

func main() {

	envsPath := flag.String("envs-path", "environments", "path to environments dir")
	gitArgs := flag.String("git-diff-args", "HEAD~ HEAD", "args in git diff")

	flag.Parse()
	args := strings.Split(*gitArgs, " ")
	args = append([]string{"diff", "--name-only"}, args...)

	buffer := bytes.Buffer{}
	cmd := exec.Command("git", args...)
	cmd.Stdout = &buffer
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		panic(fmt.Sprintf("failed to execute git: %v", err))
	}

	lines := bytes.Split(buffer.Bytes(), []byte("\n"))
	paths := make([]string, 0, len(lines))

	for i := range lines {
		path := string(lines[i])
		path = strings.TrimSpace(path)

		if path != "" {
			paths = append(paths, path)
		}
	}

	dirinfo, err := os.ReadDir(*envsPath)
	if err != nil {
		panic(fmt.Sprintf("failed to list dirinfo: %v", err))
	}

	envs := make([]string, 0, len(dirinfo))

	for i := range dirinfo {
		if dirinfo[i].IsDir() {
			envs = append(envs, dirinfo[i].Name())
		}
	}

	path := FindOptimalPath(*envsPath, paths, envs)
	fmt.Println(path)
}
