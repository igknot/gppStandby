package main

import (
	"fmt"
	"github.com/igknot/gppStandby/database"
	"github.com/jasonlvhit/gocron"
	"log"

	"database/sql"
	"github.com/igknot/gppStandby/alerting"
	"github.com/igknot/gppStandby/fileChecks"
	"github.com/igknot/gppStandby/remote"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

var db *sql.DB
var day0Date, day1Date, xxxDate, za1Date string
var globalDateStatus, za1DateStatus, statusReleaseWarehousedPayments string
var edoResponseSAPStatus, edoResponseLEGStatus, edoTrackingFileStatus string
var day1_WAITSCHEDSUBBATCH, day0_SCHEDULE, mpWaitCount, edoTrackingFileCount, edoFilesOutGoingCount, edoFilesOutGoingArchivedCount, edoResponseSAPCount, edoResponseLEGCount int64

func main() {
	info, err := os.Stat("/go/bin/gppStandby")
	var version string
	if err != nil {
		version = "version not found"
	} else {

		a := info.ModTime()
		timeFormat := "Mon Jan 2 15:04:05 MST 2006"
		version = a.Format(timeFormat)

	}

	alerting.Info("Starting Automated Standby version " + version)

	go handleRequests()

	reset()

	scheduler := gocron.NewScheduler()

	//	scheduler.Every(15).Minute().Do("checkFailureFolders")
	scheduler.Every(15).Minutes().Do(checkFailureFolders)

	scheduler.Every(1).Day().At("23:28").Do(reset)

	//1 ZA date rollover 23:32
	scheduler.Every(1).Day().At("23:33").Do(getRolloverdate, "***")

	//2 verify that tracknig file has been recieved
	scheduler.Every(1).Day().At("00:15").Do(edoTrackingFileSAPLEG)

	//3 global date rollover 00:22
	scheduler.Every(1).Day().At("00:22").Do(getRolloverdate, "ZA1")

	//4 new transactions
	scheduler.Every(1).Day().At("00:30").Do(getWAITSCHEDSUBBATCHcount)

	//5 Scheduled 00:08
	// 00:08 (00:01) track should be 0 and scheduled should be value of old_tracking after
	//scheduled and wait scheduled  will add up to  newandtracked
	scheduler.Every(1).Day().At("00:32").Do(getSCHEDULEcount)

	//6 Tracking  transactions

	scheduler.Every(1).Day().At("00:40").Do(getMPWAITcount)
	scheduler.Every(1).Day().At("00:52").Do(getMPWAITcount)

	//7 Check edo files 00:57 + 2minutes for safety
	scheduler.Every(1).Day().At("00:57").Do(checkEdoFilesOutGoing)

	//8 edoFileArchived() //01:01//
	scheduler.Every(1).Day().At("01:03").Do(edoFilesOutGoingArchived)

	//9 edoResponse anytime before 01:30 or 02:30 send mail to rcop if they are not there
	scheduler.Every(1).Day().At("01:28").Do(edoResponseSAP)
	//scheduler.Every(1).Day().At("01:29").Do(edoResponseLEG)
	//9 edoResponse anytime before 01:30 or 02:30 send mail to rcop if they are not there
	scheduler.Every(1).Day().At("02:07").Do(edoResponseSAP)
	scheduler.Every(1).Day().At("02:08").Do(edoResponseLEG)

	scheduler.Every(1).Day().At("02:15").Do(buildMailMessage)

	log.Println("creating scheduler")
	<-scheduler.Start()

}

func setDates() {
	defaultFormat := "02/Jan/2006"
	//Mon Jan 2 15:04:05 MST 2006
	timeFormat := "1504"
	nou := time.Now().Format(timeFormat)
	nouInt, _ := strconv.Atoi((nou))
	if nouInt > 2320 {
		day1Date = time.Now().AddDate(0, 0, 1).Format(defaultFormat)
		day0Date = time.Now().Format(defaultFormat)
	} else {

		day0Date = time.Now().AddDate(0, 0, -1).Format(defaultFormat)
		day1Date = time.Now().Format(defaultFormat)
	}
}

func testchecks() {

	log.Println("testchecks start")
	//checkFailureFolders()
	//getRolloverdate("ZA1")
	//getRolloverdate("***")
	//getWAITSCHEDSUBBATCHcount()
	//edoTrackingFileSAPLEG()
	//getMPWAITcount()
	//getSCHEDULEcount()
	//////
	//checkEdoFilesOutGoing() //00:57
	//edoFilesOutGoingArchived()
	////
	//edoResponseSAP() //anytime before 01:30 or 02:30 send mail to rcop if they are not there
	//edoResponseLEG()
	//buildMailMessage()
	log.Println("testchecks complete")
}

func reset() {

	log.Println("reset() start")

	day0Date, day1Date, xxxDate, za1Date = "", "", "", ""
	globalDateStatus, za1DateStatus, statusReleaseWarehousedPayments = "unset", "unset", "unset"
	day1_WAITSCHEDSUBBATCH, day0_SCHEDULE, mpWaitCount, edoTrackingFileCount = 0, 0, 0, 0
	edoFilesOutGoingCount, edoFilesOutGoingArchivedCount, edoResponseSAPCount, edoResponseLEGCount = 0, 0, 0, 0
	edoResponseSAPStatus, edoResponseLEGStatus, edoTrackingFileStatus = "Not received", "Not received", "Not received"

	setDates()
	log.Println("reset() complete")

}

func getRolloverdate(office string) {
	db = database.NewConnection()
	var message string
	defer db.Close()

	defaultFormat := "2006-01-02"
	tomorrow := time.Now().AddDate(0, 0, 1).Format(defaultFormat)
	today := time.Now().Format(defaultFormat)
	log.Println("Date roll over check:")
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

		log.Println("OFFICE:", office, "  BUSINESS_DATE ", bsnessdate[:10], "  TIME Changed ", time_stamp)
		if office == "***" {
			xxxDate = bsnessdate[:10]
			if bsnessdate[:10] != tomorrow {
				message = fmt.Sprintf("Date for office global did not roll\t Expected: %s \t Found:    %s", tomorrow, xxxDate)
				globalDateStatus = "Automatic roll failed"
			} else {
				message = fmt.Sprintf("BUSINESS_DATE for GLOBAL automatically rolled \n%s", xxxDate)
				globalDateStatus = "Rolled automatically"
			}
		}
		if office == "ZA1" {
			za1Date = bsnessdate[:10]
			if bsnessdate[:10] != today {
				message = fmt.Sprintf("BUSINESS_DATE for ZA1 automatically rolled \n%s", tomorrow, za1Date)
				za1DateStatus = "Automic roll failed"
			} else {
				message = fmt.Sprintf("Date roll for office ZA1 complete now :   %s", za1Date)
				za1DateStatus = "Rolled automatically"
			}
		}
		log.Println(message)
		alerting.Info(message)

	}
	log.Println("getRolloverdate() complete")
}

func getWAITSCHEDSUBBATCHcount() {
	log.Println("getWAITSCHEDSUBBATCHcount() start")

	db = database.NewConnection()
	defer db.Close()

	query := "SELECT count(*) transactions FROM gpp_sp.minf WHERE p_msg_sts = 'WAITSCHEDSUBBATCH' AND p_dbt_vd = '" + day1Date + "'  and p_msg_type = 'Pacs_003'"

	log.Println(query)

	rows, err := db.Query(query)
	if err != nil {
		log.Print(err.Error())
	}
	for rows.Next() {
		var transactions int64

		rows.Scan(&transactions)

		message := fmt.Sprintf("New transactions(WAITSCHEDSUBBATCH) \n %s : %d ", day1Date, transactions)
		day1_WAITSCHEDSUBBATCH = transactions
		log.Println(message)
		alerting.Info(message)
	}

	log.Println("getWAITSCHEDSUBBATCHcount() complete")
}


// after 00:01
// this should check whether /cdwasha/connectdirect/incoming/EDO_DirectDebitResponse/*SAP.LEG file exists
//remedialAction := `
//If the file has not been recieved
//			Change: operations>interfaces EDO_POSTING_REQ to inactive  > save
//					operations>apply changes> interface
//			After file has been recieved
//					operations> Tasks > New day Activities  > generate posting req for EDO > EDO_NEW > execute`
func edoTrackingFileSAPLEG() {
	//-----------------------
	log.Println("edoTrackingFileSAPLEG")

	defaultFormat := "2006-01-02"
	today := time.Now().Format(defaultFormat)

	dir := "/cdwasha/connectdirect/incoming/EDO_DirectDebitResponse/archive/"

	age := "60" // minutes
	fileName := today + "*ACDEBIT.RESPONSE.LEG.SAP*"
	log.Println("filename:", fileName)
	err, found, lineCount, fileTime := fileChecks.CheckFile(fileName, dir, age)
	if err != nil {
		message := "EdoTrackingfile LEG.SAP check failed"
		log.Println(message + err.Error())
		alerting.Callout(message)
	}
	if !found {
		message := fmt.Sprintf("EdoTrackingfile LEG.SAP file  %s not found in  %s ", fileName, dir )
		alerting.Callout(message)

	} else {
		edoTrackingFileCount = int64(lineCount - 2)
		edoTrackingFileStatus = "Received at " + fileTime
		message := fmt.Sprintf("EdoTrackingfile LEG.SAP : created at %s contains %d records \n", fileTime, edoTrackingFileCount)
		if edoFilesOutGoingCount != mpWaitCount {
			message += fmt.Sprintf("\nExpected %d ", mpWaitCount)
			alerting.Callout(message)
		} else {
			alerting.Info(message)
			log.Println(message)
		}
	}

	log.Println("edoTrackingFileSAPLEG - complete")

}


func getMPWAITcount() {

	log.Println("getMPWAITcount() start")

	db = database.NewConnection()
	defer db.Close()

	query := "SELECT count(*) transactions FROM gpp_sp.minf WHERE p_msg_sts = 'MP_WAIT' AND p_dbt_vd = '" + day1Date + "'  and p_msg_type = 'Pacs_003'"

	log.Println(query)

	rows, err := db.Query(query)
	if err != nil {
		log.Print(err.Error())
	}
	for rows.Next() {
		var transactions int64

		rows.Scan(&transactions)
		mpWaitCount = transactions

	}

	message := fmt.Sprintf("Transactions waiting for EDO Postin \nMP-WAIT  %s : %d", day1Date, mpWaitCount)
	if (day1_WAITSCHEDSUBBATCH + day0_SCHEDULE) != mpWaitCount {
		message = message + fmt.Sprintf("\nRemedial action needed: Expected %d  \n ", (day1_WAITSCHEDSUBBATCH+day0_SCHEDULE))
		alerting.Callout(message)
		statusReleaseWarehousedPayments = "failed"
	} else {
		statusReleaseWarehousedPayments = "Executed automatically"
	}
	//Executed automatically

	log.Println(message)
	alerting.Info(message)

	log.Println("getMPWAITcount() complete")

}

func getSCHEDULEcount() {
	log.Println("getSCHEDULEcount() start")
	db = database.NewConnection()
	defer db.Close()

	query := "SELECT count(*) transactions FROM gpp_sp.minf WHERE p_msg_sts = 'SCHEDULE' AND p_dbt_vd = '" + day0Date + "'  and p_msg_type = 'Pacs_003'"

	log.Println(query)

	rows, err := db.Query(query)
	if err != nil {
		log.Print(err.Error())
	}
	for rows.Next() {
		var transactions int64

		rows.Scan(&transactions)

		day0_SCHEDULE = transactions

	}

	message := fmt.Sprintf("Tracking Transactions(SCHEDULE) \n%s : %d ", day0Date, day0_SCHEDULE)

	log.Println(message)
	alerting.Info(message)

	log.Println("getSCHEDULEcount() complete")

}
func checkEdoFilesOutGoing() {
	log.Println("checkEdoFilesOutGoing(")
	dir := "/cdwasha/connectdirect/outgoing/EDO_DirectDebitRequest/"
	age := "60"
	fileName := "EDO_POST*"
	err, found, lineCount, fileTime := fileChecks.CheckFile(fileName, dir, age)
	if err != nil {
		message := "checkEdoFilesOutGoing check failed"
		log.Println(message + err.Error())
		alerting.Callout(message)
	}
	if !found {
		message := "EDO_POSTING file not found in /cdwasha/connectdirect/outgoing/EDO-DirectDebitRequest/ "
		alerting.Callout(message)

	} else {
		edoFilesOutGoingCount = int64(lineCount - 2)
		message := fmt.Sprintf("EDO-POSTING file: created at %s contains %d records \n", fileTime, edoFilesOutGoingCount)
		if edoFilesOutGoingCount != mpWaitCount {
			message += fmt.Sprintf("\nExpected %d ", mpWaitCount)
			alerting.Callout(message)
		} else {
			alerting.Info(message)
			log.Println(message)
		}
	}

	log.Println("checkEdoFilesOutGoing - complete")
}

func checkFailureFolders() {
	/*

		/cdwasha/connectdirect/incoming/BOLPES_COLL/failure "
		/cdwasha/connectdirect/incoming/GCE_COLL/failure
		/cdwasha/connectdirect/incoming/GCE_MNDT/failure
		/cdwasha/connectdirect/incoming/EDO_DirectDebitResponse/failure

	*/
	//command := "find /cdwasha/connectdirect/incoming/*/failure -type f -name *.xml -ctime -29"  // -ctime -22 created less than 22 days ago
	command := "find /cdwasha/connectdirect/incoming/*/failure -type f -name *.xml -cmin -29" // -ctime -22 created less than 22 days ago

	log.Println("checkFailureFolders: ", command)
	message := ""
	output, err := remote.RemoteSsh(command)
	if err != nil {

		log.Println("error:", err.Error())
		if err.Error() == "Process exited with status 1" {
			message = "Outgoing edo file check failed "
			log.Println(message)
		}
		alerting.Callout(message)
		log.Println("message:", message)
		return
	}

	log.Println(output)
	if len(output) == 0 {
		log.Println("checkFailureFolders: no new files ")
		return
	}
	var folders string
	for k, v := range strings.Split(output, "\n") {
		subfolder := strings.Split(v, `/`)
		if len(subfolder) > 4 {

			folders = folders + " " + subfolder[4]
		}
		log.Println(k, v)
	}
	alerting.Callout("New files in failure folder:" + folders)

}
func edoFilesOutGoingArchived() {
	// CheckFile(filename, directory , age  string ) (err error, found bool, lineCount int, fileTime string)
	dir := "/cdwasha/connectdirect/outgoing/EDO_DirectDebitRequest/archive"
	age := "60"
	fileName := "EDO_POST*"
	err, found, lineCount, fileTime := fileChecks.CheckFile(fileName, dir, age)
	if err != nil {
		message := "edoFilesOutGoingArchive check failed"
		log.Println(message + err.Error())
		alerting.Callout(message)
	}
	if !found {
		message := "EDO_POSTING file not found in /cdwasha/connectdirect/outgoing/EDO-DirectDebitRequest/archive "
		alerting.Callout(message)
		return
	}
	edoFilesOutGoingArchivedCount = int64(lineCount - 2)
	message := fmt.Sprintf("Archived EDO-POSTING file: created at %s contains %d records \n", fileTime, edoFilesOutGoingArchivedCount)
	if edoFilesOutGoingArchivedCount != mpWaitCount {
		message += fmt.Sprintf("\nExpected %d ", mpWaitCount)
		alerting.Callout(message)
	} else {
		alerting.Info(message)
		log.Println(message)
	}

}

func edoResponseLEG() {

	if edoResponseLEGStatus != "Not received" {
		log.Println("Already " + edoResponseLEGStatus)
		return
	}
	defaultFormat := "2006-01-02"
	today := time.Now().Format(defaultFormat)

	dir := "/cdwasha/connectdirect/incoming/EDO_DirectDebitResponse/archive/"
	age := "60"
	fileName := today + "*ACDEBIT.RESPONSE.LEG.2*"
	err, found, lineCount, fileTime := fileChecks.CheckFile(fileName, dir, age)
	if err != nil {
		message := "edoResponseLEG check failed"
		log.Println(message + err.Error())
		alerting.Callout(message)
	}
	if !found {
		message := "EDO Legacy response file not found in /cdwasha/connectdirect/incoming/EDO_DirectDebitResponse/archive/ "
		alerting.Callout(message)
		return
	}
	edoResponseLEGCount = int64(lineCount - 2)
	edoResponseLEGStatus = "Recieved at " + fileTime
	message := fmt.Sprintf("Archived Legacy Respomse file: recievedd at %s contains %d records \n", fileTime, edoResponseLEGCount)

	alerting.Info(message)
	log.Println(message)
}

func edoResponseSAP() {

	log.Println("EdoResponseSAP")
	if edoResponseSAPStatus != "Not received" {
		log.Println("Already " + edoResponseSAPStatus)
		return
	}
	defaultFormat := "2006-01-02"

	today := time.Now().Format(defaultFormat)

	command := "wc -l /cdwasha/connectdirect/incoming/EDO_DirectDebitResponse/archive/" + today + "*ACDEBIT.RESPONSE.SAP.*"
	log.Println(command)
	var message string
	output, err := remote.RemoteSsh(command)
	///
	if err != nil {

		log.Println("error:", err.Error())
		if err.Error() == "Process exited with status 1" {
			message = "EDO Response SAP file check failed "
			log.Println(message)
		}
		alerting.Callout(message)
		log.Println(message)
		return
	}

	log.Println(output)
	outputSlice := strings.Split(output, " ")
	linecount, _ := strconv.Atoi(outputSlice[0])
	if linecount == 0 {
		message = "EDO Response SAP file not found in /cdwasha/connectdirect/incoming/EDO-DirectDebitResponse/archive/"
		alerting.Callout(message)
		return
	}
	records := linecount - 2
	outputSlice = strings.Split(output, ".")
	recieved := outputSlice[len(outputSlice)-1]
	recievedat := recieved[0:2] + ":" + recieved[2:4] + ":" + recieved[4:6]
	edoResponseSAPStatus = "Received at " + recievedat
	edoResponseSAPCount = int64(records)
	message = fmt.Sprintf("SAP Response file contains %d records \n%s", edoResponseSAPCount, edoResponseSAPStatus)

	alerting.Info(message)
	log.Println(message)

	log.Println("EdoResponseSAP() complete ")
}

func allStatuses() {
	log.Println("start: allStatuses")
	db = database.NewConnection()
	defer db.Close()

	days := "'" + day0Date + "','" + day1Date + "'"

	query := `SELECT
	    p_dbt_vd datum,
		p_msg_sts status,
		p_msg_type msg_type,
		COUNT(*) transactions
	FROM
	gpp_sp.minf
	WHERE
	P_DBT_VD in (` + days + `)
	and p_msg_type = 'Pacs_003'

	GROUP BY
	p_dbt_vd,
		p_msg_sts ,
		p_msg_type
	ORDER BY
	p_dbt_vd DESC`

	rows, err := db.Query(query)
	if err != nil {
		log.Print(err.Error())
	}

	for rows.Next() {
		var datum string
		var status string
		var msg_type string
		var transactions int64

		rows.Scan(&datum, &status, &msg_type, &transactions)

		log.Printf("%-10s  -  %-20s  -  %-10s  -  %5d \n", datum[:10], status, msg_type, transactions)
	}

	log.Println("Complete: allStatuses")
}

func buildMailMessage() {
	subject := os.Getenv("ENVIRONMENT") + " Incoming Collections for monitoring " + day1Date

	message := "Hi \n "
	message += "\t 1. Global Office Date Roll over 23:30 –" + globalDateStatus + "\n"
	message += "\t 2. ZA1 Date Roll over 00:20 –" + za1DateStatus + "\n"
	message += fmt.Sprintf("\t 3. EDO Night Tracking responses 00:01  : %s with %d transactions ", edoTrackingFileStatus, edoTrackingFileCount)
	message += "\n\t 3. Release Warehoused Payments 00:35"
	message += "\n\t\t\t  i.      Check automatic execution – " + statusReleaseWarehousedPayments
	message +=
		fmt.Sprintf("\n\t\t\t ii.      Check Pacs.003 move to MP Wait –  :  %d new transactions processed to move to MP_WAIT; %d transactions in tracking", day1_WAITSCHEDSUBBATCH, day0_SCHEDULE)
	message +=
		fmt.Sprintf("\n\t 4. Generate EDO file – 00:55 (Ideal) – File automatically created and sent to EDO with %d transactions", edoFilesOutGoingCount)
	message +=
		fmt.Sprintf("\n\t 5. EDO responses: ")
	message +=
		fmt.Sprintf("\n\t\t For SAP     :  %s containing %d records", edoResponseSAPStatus, edoResponseSAPCount)
	message +=
		fmt.Sprintf("\n\t\t For Legacy:  %s containing %d records", edoResponseLEGStatus, edoResponseLEGCount)
	message += " \n\n Thanks \n Your friendly bot"

	alerting.SendMail(subject, message)

}

func handleRequests() {

	log.Println("Listening")
	http.HandleFunc("/test", handleTestCheck)

	log.Fatal(http.ListenAndServe(":8080", nil))
}
func handleTestCheck(w http.ResponseWriter, r *http.Request) {
	//https://medium.com/doing-things-right/pretty-printing-http-requests-in-golang-a918d5aaa000

	functions := []string{"testchecks", "getRolloverdate", "getWAITSCHEDSUBBATCHcount", "edoTrackingFileSAPLEG",
		"getMPWAITcount", "getSCHEDULEcount", "checkEdoFilesOutGoing", "checkFailureFolders", "edoFilesOutGoingArchived",
		"edoResponseLEG", "edoResponseSAP", "allStatuses", "buildMailMessage"}

	var parm string
	if parm = r.URL.Query().Get("function"); parm == "" {
		for _, f := range functions {
			url := fmt.Sprintf("%v%v%v%v", `http://`, r.Host, "/test?function=", f)
			log.Println("URL:>", url)
			fmt.Fprintf(w, "<a href=%v>%v</a></br>", url, url)
		}
	}

	log.Println("parm:", parm)

	switch parm {
	case "testchecks":
		testchecks()
	case "getRolloverdate":
		getRolloverdate("ZA1")
	case "getWAITSCHEDSUBBATCHcount":
		getWAITSCHEDSUBBATCHcount()
	case "edoTrackingFileSAPLEG":
		edoTrackingFileSAPLEG()
	case "getMPWAITcount":
		getMPWAITcount()
	case "getSCHEDULEcount":
		getSCHEDULEcount()
	case "checkEdoFilesOutGoing":
		checkEdoFilesOutGoing()
	case "checkFailureFolders":
		checkFailureFolders()
	case "edoFilesOutGoingArchived":
		edoFilesOutGoingArchived()
	case "edoResponseLEG":
		edoResponseLEG()
	case "edoResponseSAP":
		edoResponseSAP()
	case "allStatuses":
		allStatuses()
	case "buildMailMessage":
		buildMailMessage()

	}

}
