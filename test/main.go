package main

import (

	"github.com/igknot/gppStandby/alerting"

	"net/http"
	"strings"
	"fmt"
	"io/ioutil"
)

func main()  {
	//chatid := "3487598672"
	message := "this"
	//
	//url := os.Getenv("ALERT_ENDPOINT") + os.Getenv("CHAT_ID")
	//log.Print(url, message)
	resp, err := http.Post("http://go2hal.legion.sbsa.local/api/alert/3487598672", "text/plain", strings.NewReader(message))

	if err != nil {
		fmt.Println("Call to hal did not work")
	} else {defer resp.Body.Close()
		status := resp.Status


		body ,err := ioutil.ReadAll(resp.Body)
		if err != nil {
			fmt.Println("This did not work ")
		}
		fmt.Println(status, body) }
	alerting.Info(message)






}


