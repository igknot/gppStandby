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
)

//invoke callout with
func Callout(message string) {
	message = "*INVOKE - CALL OUT*\n" + message
	Info(message)


}
//send telegram

//send mail

func Info(message string) {

	url := os.Getenv("ALERT_ENDPOINT") + os.Getenv("CHAT_ID")
	 // clear special characters from message - It seems to cause hal displeasure
	reg, err := regexp.Compile("[^a-zA-Z0-9- :\n\t/]+")
	if err != nil {
		log.Fatal(err)
	}
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
	mailto := os.Getenv("MAILTO")
	if mailto == "" {
		panic("MAILTO environment variable not set")
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
	c.Rcpt(mailto)

	data, err := c.Data()
	if err != nil {
		panic(err)
	}
	defer data.Close()

	boundary := "d835e53b6b161cff115c5aaced91d1407779efa3844811da6eb831b6789b2a9a"
	defaultFormat := "2006-01-02"
	t := time.Now().Format(defaultFormat)

	fmt.Fprintf(data, "Subject: %s %s\n", subject, t)
	fmt.Fprintf(data, "MIME-Version: 1.0\n")
	fmt.Fprintf(data, "Content-Type: multipart/mixed; boundary=%s\n", boundary)

	fmt.Fprintf(data, "\n--%s\n", boundary)
	fmt.Fprintf(data, "Content-Type: text/plain; charset=utf-8\n\n")
	fmt.Fprintf(data, message,"\n")



	//fmt.Fprintf(data, "--%s--\n", boundary)
	log.Println("Mail sent to " + mailto)
}

