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
var day1_WAITSCHEDSUBBATCH, day0_SCHEDULE, mpWaitCount int
var edoResponseSAPfile, edoResponseLEGfile, edoTrackingSAPLEGfile, edoOutgoingfile, edoOutgoingfileArchived fileChecks.FileDetail

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

	scheduler.Every(15).Minutes().Do(checkFailureFolders)

	scheduler.Every(1).Day().At("23:28").Do(reset)

	//1 ZA date rollover 23:32
	scheduler.Every(1).Day().At("23:33").Do(getRolloverdate, "***")

	//2 verify that tracknig file has been recieved
	scheduler.Every(1).Day().At("00:15").Do(edoTrackingSAPLEG)

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
	scheduler.Every(1).Day().At("01:03").Do(checkEdoFilesOutGoingArchived)

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

func reset() {

	log.Println("reset() start")

	edoResponseSAPfile = fileChecks.FileDetail{
		FileName:      "*ACDEBIT.RESPONSE.SAP.2*",
		DirectoryName: "/cdwasha/connectdirect/incoming/EDO_DirectDebitResponse/archive/",
		Status:        "Not received",
		LineCount:     0,
		CreationTime:  "",
		AgeInMinutes:  "60",
		Found:         false,
		Detail:        "",
	}

	edoResponseLEGfile = fileChecks.FileDetail{
		FileName:      "*ACDEBIT.RESPONSE.LEG.2*",
		DirectoryName: "/cdwasha/connectdirect/incoming/EDO_DirectDebitResponse/archive/",
		Status:        "Not received",
		LineCount:     0,
		CreationTime:  "",
		AgeInMinutes:  "60",
		Found:         false,
	}

	edoTrackingSAPLEGfile = fileChecks.FileDetail{
		FileName:      "*ACDEBIT.RESPONSE.LEG.2*",
		DirectoryName: "/cdwasha/connectdirect/incoming/EDO_DirectDebitResponse/archive/",
		Status:        "Not received",
		LineCount:     0,
		CreationTime:  "",
		AgeInMinutes:  "60",
		Found:         false,
	}
	edoOutgoingfile = fileChecks.FileDetail{
		FileName:      "EDO_POST*",
		DirectoryName: "/cdwasha/connectdirect/outgoing/EDO_DirectDebitRequest/",
		Status:        "Not received",
		LineCount:     0,
		CreationTime:  "",
		AgeInMinutes:  "60",
		Found:         false,
	}
	edoOutgoingfileArchived = fileChecks.FileDetail{
		FileName:      "EDO_POST*",
		DirectoryName: "/cdwasha/connectdirect/outgoing/EDO_DirectDebitRequest/archive",
		Status:        "Not received",
		LineCount:     0,
		CreationTime:  "",
		AgeInMinutes:  "60",
		Found:         false,
	}

	day0Date, day1Date, xxxDate, za1Date = "", "", "", ""
	globalDateStatus, za1DateStatus, statusReleaseWarehousedPayments = "unset", "unset", "unset"
	day1_WAITSCHEDSUBBATCH, day0_SCHEDULE, mpWaitCount = 0, 0, 0

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
				message = fmt.Sprintf("BUSINESS_DATE for ZA1 not roled ZA1: %s \n Expected: %s",  za1Date,tomorrow, )
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
		var transactions int

		rows.Scan(&transactions)

		message := fmt.Sprintf("New transactions(WAITSCHEDSUBBATCH) \n %s : %d ", day1Date, transactions)
		day1_WAITSCHEDSUBBATCH = transactions
		log.Println(message)
		alerting.Info(message)
	}

	log.Println("getWAITSCHEDSUBBATCHcount() complete")
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
		var transactions int

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
		var transactions int

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
	defer log.Println("checkEdoFilesOutGoing - complete")

	if edoOutgoingfile.Found {
		return
	}
	if err := edoOutgoingfile.CheckFileCreationTime(); err != nil {
		alerting.Callout("EDO Outgoing check failed " + err.Error())
		return
	}
	if err := edoOutgoingfile.CheckFileLength(); err != nil {
		alerting.Callout("EDO Outgoing length  check failed " + err.Error())
		return
	}
	if !edoOutgoingfile.Found {
		alerting.Callout("EDO Outgoing file not found ")
		return
	}
	message := fmt.Sprintf("EDO outgoing file: created at %s contains %d records \n", edoOutgoingfile.CreationTime, edoOutgoingfile.LineCount-2)
	edoOutgoingfile.Status = "Created at " + edoOutgoingfile.CreationTime

	if (edoOutgoingfile.LineCount - 2) != mpWaitCount {
		message += fmt.Sprintf("\nExpected %d ", mpWaitCount)
		alerting.Callout(message)
		log.Println(message)
	} else {
		alerting.Info(message)
		log.Println(message)
	}

}

func checkEdoFilesOutGoingArchived() {
	log.Println("checkEdoFilesOutGoingArchived(")
	defer log.Println("checkEdoFilesOutGoingArchived - complete")

	if edoOutgoingfileArchived.Found {
		return
	}
	if err := edoOutgoingfileArchived.CheckFileCreationTime(); err != nil {
		alerting.Callout("EDO Outgoing archived check failed " + err.Error())
		return
	}
	if err := edoOutgoingfileArchived.CheckFileLength(); err != nil {
		alerting.Callout("EDO Outgoing Archived length  check failed " + err.Error())
		return
	}
	if !edoOutgoingfileArchived.Found {
		alerting.Callout("EDO Outgoing archived file not found ")
		return
	}
	message := fmt.Sprintf("Edo outgoing archived file: created at %s contains %d records \n", edoOutgoingfileArchived.CreationTime, edoOutgoingfileArchived.LineCount-2)
	edoOutgoingfileArchived.Status = "Created at " + edoOutgoingfileArchived.CreationTime

	if (edoOutgoingfileArchived.LineCount - 2) != mpWaitCount {
		message += fmt.Sprintf("\nExpected %d ", mpWaitCount)
		alerting.Callout(message)
		log.Println(message)
	} else {
		alerting.Info(message)
		log.Println(message)
	}

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
	var m map[string]int

	for k, v := range strings.Split(output, "\n") {
		subfolder := strings.Split(v, `/`)
		if len(subfolder) > 4 {
			log.Printf("Failfolder: %s" ,  v)
			m[subfolder[4]] = 1
		}
		log.Println(k, v)
	}
	var folders string
	for k,_ := range m {
		folders = folders + k
	}

	alerting.Callout("New files in failure folder:" + folders)

}

func edoTrackingSAPLEG() {
	// after 00:01
	// this should check whether /cdwasha/connectdirect/incoming/EDO_DirectDebitResponse/*SAP.LEG file exists
	//remedialAction := `
	//If the file has not been recieved
	//			Change: operations>interfaces EDO_POSTING_REQ to inactive  > save
	//					operations>apply changes> interface
	//			After file has been recieved
	//					operations> Tasks > New day Activities  > generate posting req for EDO > EDO_NEW > execute`

	log.Println("edoTrackingSAPLEG")
	defer log.Println("edoTrackingSAPLEG")
	if edoTrackingSAPLEGfile.Found {
		return
	}

	if err := edoTrackingSAPLEGfile.CheckFileCreationTime(); err != nil {
		alerting.Callout("EDO Tracking response check failed " + err.Error())
		return
	}
	if err := edoTrackingSAPLEGfile.CheckFileLength(); err != nil {
		alerting.Callout("EDO Tracking response length  check failed " + err.Error())
		return
	}
	if !edoTrackingSAPLEGfile.Found {
		alerting.Callout("EDO tracking response file not found ")
		return
	}
	message := fmt.Sprintf("LEG.SAP Tracking file: recieved at %s contains %d records \n", edoTrackingSAPLEGfile.CreationTime, edoTrackingSAPLEGfile.LineCount-2)
	edoTrackingSAPLEGfile.Status = "Received at " + edoTrackingSAPLEGfile.CreationTime
	alerting.Info(message)
	log.Println(message)

}

func edoResponseLEG() {
	if edoResponseLEGfile.Found {
		return
	}
	if err := edoResponseLEGfile.CheckFileCreationTime(); err != nil {
		alerting.Callout("EDO LEG response check failed " + err.Error())
		return
	}
	if err := edoResponseLEGfile.CheckFileLength(); err != nil {
		alerting.Callout("EDO LEG response length  check failed " + err.Error())
		return
	}
	if !edoResponseLEGfile.Found {
		alerting.Callout("EDO LEG response file not found ")
		return
	}
	message := fmt.Sprintf("LEG Response file: recieved at %s contains %d records \n", edoResponseLEGfile.CreationTime, edoResponseLEGfile.LineCount-2)
	edoResponseLEGfile.Status = "Received at " + edoResponseLEGfile.CreationTime
	alerting.Info(message)
	log.Println(message)

}

func edoResponseSAP() {
	log.Println("edoResponseSAP")
	defer log.Println("edoResponseSAP - complete")
	if edoResponseSAPfile.Found {
		return
	}
	if err := edoResponseSAPfile.CheckFileCreationTime(); err != nil {
		alerting.Callout("EDO SAP response check failed " + err.Error())
		return
	}
	if err := edoResponseSAPfile.CheckFileLength(); err != nil {
		alerting.Callout("EDO SAP response length  check failed " + err.Error())
		return
	}
	if !edoResponseSAPfile.Found {
		alerting.Callout("EDO SAP response file not found ")
		return
	}
	message := fmt.Sprintf("SAP  Response file: recieved at %s contains %d records \n", edoResponseSAPfile.CreationTime, edoResponseSAPfile.LineCount-2)
	edoResponseSAPfile.Status = "Received at " + edoResponseSAPfile.CreationTime

	edoResponseSAPfile.Detail, _ = edoResponseSAPdetailedStatus(edoResponseSAPfile)

	message = message + edoResponseSAPfile.Detail
	alerting.Info(message)
}

func edoResponseSAPdetailedStatus(fd fileChecks.FileDetail) (detail string, err error) {
	log.Println("edoResponseSAPdetailedStatus")
	defer log.Println("edoResponseSAPdetailedStatus - complete")

	command := "find " + fd.DirectoryName + " -type f -cmin -" + fd.AgeInMinutes + " -name '" + fd.FileName + `' -exec  awk '{ print substr($1,68,2) }' {} \; `
	output, err := remote.RemoteSsh(command)
	if err != nil {
		log.Printf("error-recieved\noutput: %s \n error: %s", output, err.Error())
		return
	}
	if len(output) == 0 {
		return
	}

	complete := strings.Count(output, "00")
	rejected := strings.Count(output, "02")
	tracking := strings.Count(output, "99")
	accClosed := strings.Count(output, "12")
	accLocked := strings.Count(output, "06")

	//log.Println("OUTPUT:", output)
	detail = detail + fmt.Sprintf("Complete(00)   :  %d \n", complete)
	detail = detail + fmt.Sprintf("Rejected(02)   :  %d \n", rejected)
	detail = detail + fmt.Sprintf("Tracking(99)   :  %d \n", tracking)
	detail = detail + fmt.Sprintf("Acc closed(12) :  %d \n", accClosed)
	detail = detail + fmt.Sprintf("Acc locked(06) :  %d \n", accLocked)

	return
}

func allStatuses() {
	log.Println("start: allStatuses")
	db = database.NewConnection()
	defer db.Close()

	days := "'" + day0Date + "','" + day1Date + "'"

	query := `SELECT
	    p_dbt_vd datum, p_msg_sts status, p_msg_type msg_type, COUNT(*) transactions
	FROM gpp_sp.minf
	WHERE P_DBT_VD in (` + days + `) and p_msg_type = 'Pacs_003'
	GROUP BY p_dbt_vd, p_msg_sts , p_msg_type
	ORDER BY p_dbt_vd DESC`

	rows, err := db.Query(query)
	if err != nil {
		log.Print(err.Error())
	}

	for rows.Next() {
		var datum, status, msg_type string
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
	message += fmt.Sprintf("\t 3. EDO Night Tracking responses 00:01  : %s with %d transactions ", edoTrackingSAPLEGfile.Status, edoTrackingSAPLEGfile.LineCount-2)
	message += "\n\t 3. Release Warehoused Payments 00:35"
	message += "\n\t\t\t  i.      Check automatic execution – " + statusReleaseWarehousedPayments
	message +=
		fmt.Sprintf("\n\t\t\t ii.      Check Pacs.003 move to MP Wait –  :  %d new transactions processed to move to MP_WAIT; %d transactions in tracking", day1_WAITSCHEDSUBBATCH, day0_SCHEDULE)
	message +=
		fmt.Sprintf("\n\t 4. Generate EDO file – 00:55 (Ideal) – File automatically created and sent to EDO with %d transactions", edoOutgoingfile.LineCount-2)
	message +=
		fmt.Sprintf("\n\t 5. EDO responses: ")
	message +=
		fmt.Sprintf("\n\t\t For SAP     :  %s containing %d records", edoResponseSAPfile.Status, edoResponseSAPfile.LineCount-2)
	message +=
		fmt.Sprintf("\n\t\t For Legacy:  %s containing %d records", edoResponseLEGfile.Status, edoResponseLEGfile.LineCount-2)
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
	functions := []string{
		"setDates",
		"reset",
		"getRolloverdate-ZA1",
		"getRolloverdate-global",
		"getWAITSCHEDSUBBATCHcount",
		"getMPWAITcount",
		"getSCHEDULEcount",
		"checkEdoFilesOutGoing",
		"checkEdoFilesOutGoingArchived",
		"checkFailureFolders",
		"edoTrackingSAPLEG",
		"edoResponseLEG",
		"edoResponseSAP",
		"allStatuses",
		"buildMailMessage",

	}

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

	case "setDates" :
		setDates()
	case "getRolloverdate-ZA1":
		getRolloverdate("ZA1")
	case "getRolloverdate-global":
		getRolloverdate("***")
	case "reset":
		reset()
	case "getWAITSCHEDSUBBATCHcount":
		getWAITSCHEDSUBBATCHcount()
	case "getMPWAITcount":
		getMPWAITcount()
	case "getSCHEDULEcount":
		getSCHEDULEcount()
	case "checkEdoFilesOutGoing":
		checkEdoFilesOutGoing()
	case "checkFailureFolders":
		checkFailureFolders()
	case "checkEdoFilesOutGoingArchived":
		checkEdoFilesOutGoingArchived()
	case "edoTrackingFileSAPLEG":
		edoTrackingSAPLEG()
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
