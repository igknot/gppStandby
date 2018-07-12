package fileChecks

import (
	"log"

	"github.com/igknot/gppStandby/remote"
	"strconv"
	"strings"
)

func CheckFile(filename, directory, age string) (err error, found bool, lineCount int, fileTime string) {

	command := "find " + directory + " -type f -cmin -" + age + " -name '" + filename + `' -exec wc -l {} \; `

	output, err := remote.RemoteSsh(command)
	if err != nil {

		if err.Error() == "Process exited with status 1" {
			found = false
			err = nil
		} else {
			log.Printf("error-recieved\noutput: %s \n error: %s", output, err.Error())
		}
		return
	}
	if len(output) == 0 {
		found = false
		return
	}

	outputSlice := strings.Split(output, " ")
	lineCount, _ = strconv.Atoi(outputSlice[0])

	command = "find " + directory + " -type f -cmin -" + age + " -name '" + filename + `' -exec ls -l {} \; `

	output, err = remote.RemoteSsh(command)
	if err != nil {

		if err.Error() == "Process exited with status 1" {
			found = false
			err = nil
		} else {
			log.Printf("error-recieved\noutput: %s \n error: %s", output, err.Error())
		}
		return
	}
	found = true

	outputSlice = strings.Split(output, " ")
	fileTime = outputSlice[7]

	return

}
