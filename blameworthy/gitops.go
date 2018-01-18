package blameworthy

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

const hashLength = 16		// number of hash characters to preserve

type FileHistory []FileCommit;

type FileCommit struct {
	Hash string
	Hunks []Hunk
}

type Hunk struct {
	OldStart int
	OldLength int
	NewStart int
	NewLength int
}

func RunGitLog(repository_path string) (io.ReadCloser, error) {
	cmd := exec.Command("/usr/bin/git",
		"-C", repository_path,
		"log",
		"-U0",
		"--format=commit %H",
		"--no-prefix",
		"--no-renames",
		"--reverse",

		// Avoid invoking custom diff commands or conversions.
		"--no-ext-diff",
		"--no-textconv",

		// Treat a merge as a simple diff against its 1st parent:
		"--first-parent",
		"-m",
	)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	//defer cmd.Wait()  // drat, when will we do this?
	err = cmd.Start()
	if err != nil {
		return nil, err
	}
	return stdout, nil
}

// Given an input stream from `git log`, print out an abbreviated form
// of the log that is missing the "+" and "-" lines that give the actual
// content of each diff.  Each line like "@@ -0,0 +1,3 @@" introducing
// content will have its final double-at suffixed with a dash (like
// this: "@@-") so blameworthy will recognize that the content has been
// omitted when it reads the log as input.
func StripGitLog(input io.Reader) (error) {
	re, _ := regexp.Compile(`@@ -(\d+),?(\d*) \+(\d+),?(\d*) `)

	scanner := bufio.NewScanner(input)

	const maxCapacity = 100*1024*1024
	buf := make([]byte, maxCapacity)
	scanner.Buffer(buf, maxCapacity)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "commit ") {
			fmt.Print(line + "\n")
		} else if strings.HasPrefix(line, "--- ") {
			fmt.Print(line + "\n")
		} else if strings.HasPrefix(line, "+++ ") {
			fmt.Print(line + "\n")
		} else if strings.HasPrefix(line, "@@ ") {
			rest := line[3:]
			i := strings.Index(rest, " @@")
			fmt.Printf("@@ %s @@-\n", rest[:i])

			result_slice := re.FindStringSubmatch(line)
			//oldStart, _ := strconv.Atoi(result_slice[1])
			oldLength := 1
			if len(result_slice[2]) > 0 {
				oldLength, _ = strconv.Atoi(result_slice[2])
			}
			//newStart, _ := strconv.Atoi(result_slice[3])
			newLength := 1
			if len(result_slice[4]) > 0 {
				newLength, _ = strconv.Atoi(result_slice[4])
			}
			lines_to_skip := oldLength + newLength
			for i := 0; i < lines_to_skip; i++ {
				scanner.Scan()
			}
		}
	}
	return scanner.Err()
}

func ParseGitLog(input_stream io.ReadCloser) (map[string]FileHistory, error) {
	scanner := bufio.NewScanner(input_stream)
	historyMap := make(map[string]FileHistory)

	var commitHash string
	var currentCommit *FileCommit

	// A dash after the second "@@" is a signal from our command
	// `strip-git-log` that it has removed the "+" and "-" lines
	// that would have followed next.
	re, _ := regexp.Compile(`@@ -(\d+),?(\d*) \+(\d+),?(\d*) @@(-?)`)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "commit ") {
			commitHash = line[7:7+hashLength]
		} else if strings.HasPrefix(line, "--- ") {
			path := line[4:]
			scanner.Scan()  // read the "+++" line
			if path == "/dev/null" {
				line2 := scanner.Text()
				path = line2[4:]
			}
			history, ok := historyMap[path]
			if !ok {
				history = FileHistory{}
			}
			history = append(history,
				FileCommit{commitHash, []Hunk{}})
			currentCommit = &history[len(history) - 1]
			historyMap[path] = history
		} else if strings.HasPrefix(line, "@@ ") {
			result_slice := re.FindStringSubmatch(line)
			OldStart, _ := strconv.Atoi(result_slice[1])
			OldLength := 1
			if len(result_slice[2]) > 0 {
				OldLength, _ = strconv.Atoi(result_slice[2])
			}
			NewStart, _ := strconv.Atoi(result_slice[3])
			NewLength := 1
			if len(result_slice[4]) > 0 {
				NewLength, _ = strconv.Atoi(result_slice[4])
			}

			currentCommit.Hunks = append(currentCommit.Hunks, Hunk{
				OldStart, OldLength, NewStart, NewLength,
			})

			// Expect no unified diff if hunk header ends in "@@-"
			is_stripped := len(result_slice[5]) > 0
			if ! is_stripped {
				lines_to_skip := OldLength + NewLength
				for i := 0; i < lines_to_skip; i++ {
					scanner.Scan()
				}
			}
		}
	}
	return historyMap, nil
}