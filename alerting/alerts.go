package alerting

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
)

//invoke callout with

//send telegram

//send mail

func Info(message string) {

	url := os.Getenv("ALERT_ENDPOINT") + os.Getenv("CHAT_ID")
	log.Print(url, message)
	resp, err := http.Post(url, "text/plain", strings.NewReader(message))

	if err != nil {
		fmt.Println("Call to hal did not work")
		return
	} else {
		defer resp.Body.Close()
		status := resp.Status

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			fmt.Println("This did not work ")
		}
		fmt.Println(status, body)
	}

}
