package main

import (

	"github.com/igknot/gppStandby/alerting"

	"fmt"
	"time"
	"strconv"
)

func main()  {
	telegramtests()
	//timetests()


}



func timetests(){
//Mon Jan 2 15:04:05 MST 2006
	timeFormat := "1504"
	nou := time.Now().Format(timeFormat)
	 nouInt ,_ := strconv.Atoi((nou))
	 if nouInt > 1504 {
	 	fmt.Println("later")
	 }
	 fmt.Println("nouInt:",nouInt)

}
func telegramtests() {
	//chatid := "3487598672"
	//message := "*bold text*" //
	//message := " _italic text_  [inline URL](http://www.example.com/)  [inline mention of a user](tg://user?id=123456789)  `inline fixed-width code` "
	//message :=  " outside block \n" +
	//	"```block_language " +
	//" pre-formatted fixed-width code block " +
	// " ```"
	//message := " Testing emoji u2705 \u2705 \n u26A0 \u26A0 "

	//message := " S_EE  "
	//
	//
	//
	////
	////url := os.Getenv("ALERT_ENDPOINT") + os.Getenv("CHAT_ID")
	////log.Print(url, message)
	//resp, err := http.Post("http://go2hal.legion.sbsa.local/api/alert/3487598672", "text/plain", strings.NewReader(message))
	//fmt.Println("Status : ", resp.Status)
	//if err != nil {
	//	fmt.Println("Call to hal did not work ", err.Error())
	//} else {
	//	defer resp.Body.Close()
	//	status := resp.Status
	//
	//	body, err := ioutil.ReadAll(resp.Body)
	//	if err != nil {
	//		fmt.Println("This did not work ")
	//	}
	//	fmt.Println(status, body)
	//}
	message := " Testing emoji u2705 \u2705 \n u26A0 \u26A0 "
	alerting.Callout(message)
}


