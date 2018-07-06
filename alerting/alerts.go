package alerting

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"log"
	"net/smtp"
	"time"
	"regexp"
	"bytes"
)

//invoke callout with
func Callout(message string) {


	reg, err := regexp.Compile("[^a-zA-Z0-9- :\n\t/]+")
	if err != nil {
		log.Fatal(err)
	}

	message = reg.ReplaceAllString(message, "")
	url := os.Getenv("CALLOUT_ENDPOINT") + os.Getenv("CHAT_ID")
	 log.Println("URL:>",url)


    var jsonStr = []byte(`{"message":"Please check GPP. ` + message + `","title":"gpp callout" }`)

    log.Println("call out:",string(jsonStr))

    req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonStr))
    req.Header.Set("Accept", "application/json")
    req.Header.Set("Content-Type", "application/json")

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        log.Println(err.Error())
        return
    }
    defer resp.Body.Close()

    log.Println("response Status:", resp.Status)
    log.Println("response Headers:", resp.Header)
    body, _ := ioutil.ReadAll(resp.Body)
    log.Println("response Body:", string(body))

	message = "INVOKE - CALL OUT\n" + message
	Info(message)


}


func Info(message string) {
	defaultFormat := "2006-01-02 15:04"
	infoTime := time.Now().Format(defaultFormat)
	url := os.Getenv("ALERT_ENDPOINT") + os.Getenv("CHAT_ID")
	 // clear special characters from message - It seems to cause hal displeasure
	reg, err := regexp.Compile("[^a-zA-Z0-9- :\n\t/]+")
	if err != nil {
		log.Fatal(err)
	}
	message = infoTime +"\n"+ message
	message = reg.ReplaceAllString(message, "")


	resp, err := http.Post(url, "text/plain", strings.NewReader(message))

	if err != nil {
		log.Println("Call to hal did not work")
		return
	} else {
		defer resp.Body.Close()
		status := resp.Status

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Printf("Telegram message send status:%s \nError: %s\nBody:%s",status,err.Error(),body)
		}
		log.Printf("Telegram message: % send status: %s",message,status)
	}

}

func SendMail(subject ,message string ) {
	mailfrom := os.Getenv("MAILFROM")
	if mailfrom == "" {
		log.Println("MAILFROM not set")
	}
	to := strings.Split(os.Getenv("MAILTO"),",")
	var mailto string
	if len(to) == 0 {
		log.Println("MAILTO environment variable not set")
	} else{

		log.Println("mailto: ", to)

	}
	server := os.Getenv("MAILSERVER")
	if server == "" {
		log.Println("MAILSERVER environment variable not set")
	}

	c, err := smtp.Dial(server)
	if err != nil {
		log.Println(err.Error())
	}

	c.Mail(mailfrom)
	for _, t := range to {
		log.Println("recipient:",t)
		c.Rcpt(t)
	}

	data, err := c.Data()
	if err != nil {
		log.Println(err.Error())
	}
	defer data.Close()


	defaultFormat := "2006-01-02"
	t := time.Now().Format(defaultFormat)

	fmt.Fprintf(data, "Subject: %s %s\n", subject, t)
	fmt.Fprintf(data, "MIME-Version: 1.0\n")

	fmt.Fprintf(data, "Content-Type: text/plain; charset=utf-8\n\n")
	fmt.Fprintf(data, message)

	log.Println("Mail sent to " + mailto)


}

