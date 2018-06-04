package main

import (
	"bytes"
	"database/sql"
	"fmt"
	"github.com/igknot/gppStandby/database"
	"github.com/jasonlvhit/gocron"
	"golang.org/x/crypto/ssh"
	"log"
	"os"
	//"strings"
	"io/ioutil"
	"time"
)

var db *sql.DB
var zaDate, xxxDate, edoFileName string
var newTrans, trackedTrans, scheduledTrans, newAndtracked int64



func main() {
	//edoResponseLEG()
	//edoResponseSAP()
	//
	//return


	
	logNow()
	db = database.NewConnection()
	defer db.Close()
	reset()
	//allchecks()

	scheduler := gocron.NewScheduler()
	//1 ZA date rollover 23:32
	scheduler.Every(1).Day().At("23:37").Do(getRolloverdate, "***")

	//2 global date rollover 00:22
	scheduler.Every(1).Day().At("00:22").Do(getRolloverdate, "ZA1")

	//3 new transactions
	scheduler.Every(1).Day().At("23:55").Do(newTransactions)

	//4 Tracking  transactions
	scheduler.Every(1).Day().At("23:50").Do(trackingTransactions)

	//5 Scheduled 00:08
	// 00:08 (00:01) track should be 0 and scheduled should be value of old_tracking after
	//scheduled and wait scheduled  will add up to  newandtracked
	scheduler.Every(1).Day().At("00:08").Do(ScheduledTransactions)

	//6 new and tracked 00:37
	scheduler.Every(1).Day().At("00:37").Do(newAndTracked)

	//7 Check edo files 00:57 + 2minutes for safety
	scheduler.Every(1).Day().At("00:57").Do(edoFiles)

	//8 edoFileArchived() //01:01//
	scheduler.Every(1).Day().At("01:01").Do(edoFileArchived)

	//9 edoResponse anytime before 01:30 or 02:30 send mail to rcop if they are not there
	scheduler.Every(1).Day().At("01:28").Do(edoResponseSAP)
	scheduler.Every(1).Day().At("01:28").Do(edoResponseLEG)
	//9 edoResponse anytime before 01:30 or 02:30 send mail to rcop if they are not there
	scheduler.Every(1).Day().At("02:28").Do(edoResponseSAP)
	scheduler.Every(1).Day().At("02:28").Do(edoResponseLEG)

	<-scheduler.Start()

}

func logNow() {
	defaultFormat := "2006-01-02 15:04"
	now := time.Now().Format(defaultFormat)
	fmt.Println("\n\n-------------------\n",now)

}

func allchecks() {
	fmt.Println("---------------------------------------------")
	logNow()

	//23:31 and 00:21
	getRolloverdate("ZA1")
	getRolloverdate("***")
	newTransactions()
	trackingTransactions()
	ScheduledTransactions()
	newAndTracked()
	edoFiles()        //00:57
	edoFileArchived() //01:01
	edoResponseSAP()  //anytime before 01:30 or 02:30 send mail to rcop if they are not there
	edoResponseLEG()

	fmt.Println("ZADATE: ", zaDate, "\nGLOBALDATE ", xxxDate, "\nNEW:", newTrans, "\nTracked", trackedTrans, "\nScheduled: ", scheduledTrans,
		"\nNewAndTrackedc", newAndtracked)
}
func reset() {
	fmt.Println("---------------------------------------------")
	zaDate = ""
	xxxDate = ""
	newTrans = 0
	trackedTrans = 0
	scheduledTrans = 0
	newAndtracked = 0

}
func getRolloverdate(office string) {
	db = database.NewConnection()

	defer db.Close()

	logNow()
	defaultFormat := "2006-01-02"
	tomorrow := time.Now().AddDate(0, 0, 1).Format(defaultFormat)
	today := time.Now().Format(defaultFormat)
	fmt.Println("\n\nDate roll over check:")
	query := " select office, time_stamp, bsnessdate  from gpp_sp.banks where office = '" + office + "'"
	rows, err := db.Query(query)
	if err != nil {
		log.Print(err.Error())
	}
	for rows.Next() {
		var office string
		var time_stamp string
		var bsnessdate string
		rows.Scan(&office, &time_stamp, &bsnessdate)

		fmt.Println("OFFICE:", office, "  BUSINESS_DATE ", bsnessdate[:10], "  TIME Changed ", time_stamp)
		if office == "***" {
			xxxDate = bsnessdate[:10]
			if bsnessdate[:10] != tomorrow {
				fmt.Printf("Date for office *** did not roll\nExpected: %s \nFound:    %s", tomorrow, bsnessdate[:10])
			} else {
				fmt.Println("Date roll Complete for ***")
			}
		}
		if office == "ZA1" {
			zaDate = bsnessdate[:10]
			if bsnessdate[:10] != today {
				fmt.Printf("Date for office *** did not roll\nExpected: %s \nFound:    %s", today, bsnessdate[:10])
			} else {
				fmt.Println("Date roll Complete for ZA1")
			}
		}

	}

}

func newTransactions() {
	db = database.NewConnection()
	defer db.Close()

	logNow()
	defaultFormat := "02/Jan/2006"

	tomorrow := time.Now().AddDate(0, 0, 1).Format(defaultFormat)

	query := "SELECT count(*) transactions FROM gpp_sp.minf WHERE p_msg_sts LIKE 'WAIT%' AND p_dbt_vd = '" + tomorrow + "'"

	fmt.Println("\n\n", query)

	rows, err := db.Query(query)
	if err != nil {
		log.Print(err.Error())
	}
	for rows.Next() {
		var transactions int64

		rows.Scan(&transactions)

		fmt.Println("New transactions(WAIT%): ", transactions)
		newTrans = transactions
	}

}

func trackingTransactions() {
	db = database.NewConnection()
	defer db.Close()
	logNow()
	defaultFormat := "02/Jan/2006"
	today := time.Now().Format(defaultFormat)
	//tomorrow := time.Now().AddDate(0, 0, 1).Format(defaultFormat)

	query := "SELECT count(*) transactions FROM gpp_sp.minf WHERE p_msg_sts LIKE 'MP_%' AND p_dbt_vd = '" + today + "'"

	fmt.Println("\n\n", query)

	rows, err := db.Query(query)
	if err != nil {
		log.Print(err.Error())
	}
	for rows.Next() {
		var transactions int64

		rows.Scan(&transactions)

		fmt.Println("Tracking transactions(MP_%) ", transactions)
		trackedTrans = transactions
	}

}

func ScheduledTransactions() {
	db = database.NewConnection()
	defer db.Close()
	logNow()
	defaultFormat := "02/Jan/2006"
	today := time.Now().Format(defaultFormat)

	query := "SELECT count(*) transactions FROM gpp_sp.minf WHERE p_msg_sts LIKE 'SCH%' AND p_dbt_vd = '" + today + "' "

	fmt.Println(" \n\n", query)

	rows, err := db.Query(query)
	if err != nil {
		log.Print(err.Error())
	}
	for rows.Next() {
		var transactions int64

		rows.Scan(&transactions)

		fmt.Println("Scheduled transactions ", transactions)
		scheduledTrans = transactions

	}
	fmt.Printf("Tracked - prev : %d \nScheduled: %d  \n", trackedTrans, scheduledTrans)

	if (trackedTrans) != scheduledTrans {
		fmt.Println("Alert")
	} else {
		fmt.Println(" GOOD ")
	}

}

func newAndTracked() {
	db = database.NewConnection()
	defer db.Close()
	logNow()

	defaultFormat := "02/Jan/2006"
	today := time.Now().Format(defaultFormat)

	query := "SELECT count(*) transactions FROM gpp_sp.minf WHERE p_msg_sts LIKE 'MP_WAIT%' AND p_dbt_vd = '" + today + "'  "

	fmt.Println(" \n\n", query)

	rows, err := db.Query(query)
	if err != nil {
		log.Print(err.Error())
	}
	for rows.Next() {
		var transactions int64

		rows.Scan(&transactions)

		fmt.Println("New and tracked", transactions)
		newAndtracked = transactions
	}
	if newAndtracked != (scheduledTrans + newTrans) {
		fmt.Println("new and tracked does not add up (scheduled and new ")
	}

}
func edoFiles() {
	logNow()
	command := "find /cdwasha/connectdirect/outgoing/EDO_DirectDebitRequest -type f -cmin -60 -name 'EDO_POST*' -exec wc -l {} \\; "
	// 00:57

	fmt.Println("EdoFiles\n", command)

	output, err := remoteRun(command)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	fmt.Println(output)
	fmt.Println("EdoFiles --end\n")
}

func edoFileArchived() { //01:00
	logNow()
	command := "find /cdwasha/connectdirect/outgoing/EDO_DirectDebitRequest/archive -type f -cmin -1440 -name 'EDO_POST*' -exec wc -l {} \\; "
	// 00:
	//gppadm@s1paygpp1v[/cdwasha/connectdirect/outgoing/EDO_DirectDebitRequest/ ]$
	fmt.Println("EdoFilesArchived\n", command)
	output, err := remoteRun(command)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	fmt.Println(output)
	fmt.Println("EdoFilesArchived --end\n")
}
func edoResponseLEG() {
	logNow()
	fmt.Println("EdoResponseLEG\n")
	defaultFormat := "2006-01-02"

	today := time.Now().Format(defaultFormat)

	command := "wc -l /cdwasha/connectdirect/incoming/EDO_DirectDebitResponse/archive/" + today + "*ACDEBIT.RESPONSE.LEG.*"
	//command := "find /cdwasha/connectdirect/outgoing/EDO_DirectDebitRequest -type f -cmin -1440 -name 'EDO_POST*' -exec wc -l {} \\; "
	// 00:57
	//gppadm@s1paygpp1v[/cdwasha/connectdirect/outgoing/EDO_DirectDebitRequest/ ]$
	fmt.Println( command)
	output, err := remoteRun(command)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	fmt.Println(output)
	fmt.Println("EdoResponseLEG --end\n")

}

func edoResponseSAP() {
	logNow()
	defaultFormat := "2006-01-02"

	today := time.Now().Format(defaultFormat)

	command := "wc -l /cdwasha/connectdirect/incoming/EDO_DirectDebitResponse/archive/" + today + "*ACDEBIT.RESPONSE.SAP.*"
	fmt.Println("EdoResponseSAP\n", command)
	output, err := remoteRun(command)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	fmt.Println(output)
	fmt.Println("EdoResponseSAP --end\n")
}

func sendmail() {}

func sendtelegram() {}

func callout() {}

func remoteRun(cmd string) (string, error) {

	config := &ssh.ClientConfig{
		User: sshUser(),

		Auth: []ssh.AuthMethod{
			PublicKeyFile("/tmp/id_rsa")},
	}

	config.HostKeyCallback = ssh.InsecureIgnoreHostKey()

	// Connect
	addr := sshEndpoint()
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {

		return "", err
	}
	// Create a session. It is one session per command.
	session, err := client.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()
	var b bytes.Buffer  // import "bytes"
	session.Stdout = &b // get output

	err = session.Run(cmd)

	return b.String(), err
}

func sshUser() string {
	return os.Getenv("SSH_USER")
}

func sshEndpoint() string {
	return os.Getenv("SSH_ENDPOINT")
}

func PublicKeyFile(file string) ssh.AuthMethod {
	buffer, err := ioutil.ReadFile(file)
	if err != nil {
		fmt.Println(err.Error())
		return nil
	}

	key, err := ssh.ParsePrivateKey(buffer)
	if err != nil {
		fmt.Println("Parsing failed", err.Error())
		return nil
	}

	return ssh.PublicKeys(key)
}
