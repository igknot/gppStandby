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
	//message := "*bold text*" //
	//message := " _italic text_  [inline URL](http://www.example.com/)  [inline mention of a user](tg://user?id=123456789)  `inline fixed-width code` "
	message :=  " outside block \n" +
		"```block_language " +
	" pre-formatted fixed-width code block " +
	 " ```"

	//
	//url := os.Getenv("ALERT_ENDPOINT") + os.Getenv("CHAT_ID")
	//log.Print(url, message)
	resp, err := http.Post("http://go2hal.legion.sbsa.local/api/alert/3487598672", "text/plain", strings.NewReader(message))

	fmt.Println("Status : ",resp.Status)

	if err != nil {
		fmt.Println("Call to hal did not work ")
	} else {defer resp.Body.Close()
		status := resp.Status


		body ,err := ioutil.ReadAll(resp.Body)
		if err != nil {
			fmt.Println("This did not work ")
		}
		fmt.Println(status, body) }
	alerting.Info(message)






}


