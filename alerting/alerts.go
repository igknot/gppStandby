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
	 fmt.Println("URL:>",url)


    var jsonStr = []byte(`{"message":"Please check GPP. ` + message + `","title":"gpp callout" }`)

    fmt.Printf(string(jsonStr))

    req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonStr))
    req.Header.Set("Accept", "application/json")
    req.Header.Set("Content-Type", "application/json")

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        panic(err)
    }
    defer resp.Body.Close()

    fmt.Println("response Status:", resp.Status)
    fmt.Println("response Headers:", resp.Header)
    body, _ := ioutil.ReadAll(resp.Body)
    fmt.Println("response Body:", string(body))

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
		fmt.Println("Call to hal did not work")
		return
	} else {
		defer resp.Body.Close()
		status := resp.Status

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Printf("Telegram message send status:%s \nError: %s\nBody:%s",status,err.Error(),body)
		}
		log.Print("Telegram message send status:",status)
	}

}

func SendMail(subject ,message string ) {
	mailfrom := os.Getenv("MAILFROM")
	if mailfrom == "" {
		panic("MAILFROM not set")
	}
	to := strings.Split(os.Getenv("MAILTO"),",")
	var mailto string
	if len(to) == 0 {
		panic("MAILTO environment variable not set")
	} else{

		fmt.Println("mailto: ", to)

	}
	server := os.Getenv("MAILSERVER")
	if server == "" {
		panic("MAILSERVER environment variable not set")
	}

	c, err := smtp.Dial(server)
	if err != nil {
		panic(err)
	}

	c.Mail(mailfrom)
	for _, t := range to {
		fmt.Println("recipient:",t)
		c.Rcpt(t)
	}

	data, err := c.Data()
	if err != nil {
		panic(err)
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

