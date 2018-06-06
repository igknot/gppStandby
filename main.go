package main

import (
	"database/sql"
	"fmt"
	"github.com/igknot/gppStandby/database"
	"github.com/jasonlvhit/gocron"
	"log"

	"github.com/igknot/gppStandby/alerting"
	"github.com/igknot/gppStandby/remote"
	"strconv"
	"strings"
	"time"
)

var db *sql.DB
var day0Date, day1Date, xxxDate, za1Date string

var day1_WAITSCHEDSUBBATCH, day0_SCHEDULE, day1_MP_WAIT, day0_NightTrackingFile, day1_edoPosting, day1_edoPostingArchived, day1_sapResponse, day1_legacyResponse int64

func main() {
	defaultFormat := "02/Jan/2006"
	day1Date = time.Now().AddDate(0, 0, 1).Format(defaultFormat)
	day0Date = time.Now().Format(defaultFormat)

	//edoFilesOutGoing()
	//edoFilesOutGoingArchived()
	//edoTrackingFileSAPLEG() //00:15

	//getSCHEDULEcount()
	//getWAITSCHEDSUBBATCHcount()
	//getMPWAITcount()

	//return
	//
	//

	logNow()
	db = database.NewConnection()
	defer db.Close()
	reset()
	//allchecks()

	scheduler := gocron.NewScheduler()
	scheduler.Every(10).Minutes().Do(allStatuses)

	//1 ZA date rollover 23:32
	scheduler.Every(1).Day().At("23:40").Do(getRolloverdate, "***")

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
	scheduler.Every(1).Day().At("00:37").Do(getMPWAITcount)

	//7 Check edo files 00:57 + 2minutes for safety
	scheduler.Every(1).Day().At("00:57").Do(edoFilesOutGoing)

	//8 edoFileArchived() //01:01//
	scheduler.Every(1).Day().At("01:03").Do(edoFilesOutGoingArchived)

	//9 edoResponse anytime before 01:30 or 02:30 send mail to rcop if they are not there
	scheduler.Every(1).Day().At("01:28").Do(edoResponseSAP)
	scheduler.Every(1).Day().At("01:28").Do(edoResponseLEG)
	//9 edoResponse anytime before 01:30 or 02:30 send mail to rcop if they are not there
	scheduler.Every(1).Day().At("02:00").Do(edoResponseSAP)
	scheduler.Every(1).Day().At("02:00").Do(edoResponseLEG)

	<-scheduler.Start()

}

func logNow() {
	defaultFormat := "2006-01-02 15:04"
	now := time.Now().Format(defaultFormat)
	fmt.Println("\n\n-------------------", now)

}

func allchecks() {

	logNow()

	//23:31 and 00:21
	getRolloverdate("ZA1")
	getRolloverdate("***")
	getWAITSCHEDSUBBATCHcount()
	getMPWAITcount()
	getSCHEDULEcount()
	//newAndTracked()
	edoFilesOutGoing() //00:57
	//edoFileArchived()  //01:01
	edoResponseSAP() //anytime before 01:30 or 02:30 send mail to rcop if they are not there
	edoResponseLEG()

}

func reset() {
	logNow()

}

func getRolloverdate(office string) {
	db = database.NewConnection()
	var message string
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
				message = fmt.Sprintf("Date for office *** did not roll\nExpected: %s \nFound:    %s", tomorrow, bsnessdate[:10])

			} else {
				message = fmt.Sprintf("Date roll for office *** complete now :   %s", bsnessdate[:10])
			}
		}
		if office == "ZA1" {
			za1Date = bsnessdate[:10]
			if bsnessdate[:10] != today {
				message = fmt.Sprintf("Date for office ZA1 did not roll\nExpected: %s \nFound:    %s", tomorrow, bsnessdate[:10])
			} else {
				message = fmt.Sprintf("Date roll for office ZA1 complete now :   %s", bsnessdate[:10])
			}
		}
		fmt.Println(message)
		alerting.Info(message)

	}

}

func getWAITSCHEDSUBBATCHcount() {
	db = database.NewConnection()
	defer db.Close()

	logNow()

	query := "SELECT count(*) transactions FROM gpp_sp.minf WHERE p_msg_sts = 'WAITSCHEDSUBBATCH' AND p_dbt_vd = '" + day1Date + "'  and p_msg_type = 'Pacs_003'"

	fmt.Println("\n\n", query)

	rows, err := db.Query(query)
	if err != nil {
		log.Print(err.Error())
	}
	for rows.Next() {
		var transactions int64

		rows.Scan(&transactions)

		message := fmt.Sprintf("New transactions \nWAITSCHEDSUBBATCH - %s : %d ", day1Date, transactions)
		day1_WAITSCHEDSUBBATCH = transactions
		fmt.Println(message)
		alerting.Info(message)
	}

}

func edoTrackingFileSAPLEG() {
	// after 00:01
	// this should check whether /cdwasha/connectdirect/incoming/EDO_DirectDebitResponse/*SAP.LEG file exists
	//remedialAction := `
	//If the file has not been recieved
	//			Change: operations>interfaces EDO_POSTING_REQ to inactive  > save
	//					operations>apply changes> interface
	//			After file has been recieved
	//					operations> Tasks > New day Activities  > generate posting req for EDO > EDO_NEW > execute`
	logNow()
	fmt.Println("edoTrackingFileSAPLEG\n")
	defaultFormat := "2006-01-02"

	today := time.Now().Format(defaultFormat)

	command := "wc -l /cdwasha/connectdirect/incoming/EDO_DirectDebitResponse/" + today + "*ACDEBIT.RESPONSE.LEG.SAP*"

	var message string

	fmt.Println(command)
	remote.RemoteSsh(command)
	output, err := remote.RemoteSsh(command)
	if err != nil {
		fmt.Println("error-recieved\noutput:", output)
		fmt.Println("error:", err.Error())
		if err.Error() == "Process exited with status 1" {
			fmt.Println("No file recieved")
			message = "Night tracking file not recieved :  "
		}
		alerting.Callout(message)
		fmt.Println(message)
		return
	}

	fmt.Println(output)
	outputSlice := strings.Split(output, " ")
	linecount, _ := strconv.Atoi(outputSlice[0])
	records := linecount - 2
	day0_NightTrackingFile = int64(records)
	message = fmt.Sprintf("Tracking file recived from EDO contains %d records \n", day0_NightTrackingFile)
	alerting.Info(message)
	fmt.Println(message)
	fmt.Println("\nedoTrackingFileSAPLEG --end\n")

}

func getMPWAITcount() {

	//

	db = database.NewConnection()
	defer db.Close()
	logNow()

	query := "SELECT count(*) transactions FROM gpp_sp.minf WHERE p_msg_sts = 'MP_WAIT' AND p_dbt_vd = '" + day1Date + "'  and p_msg_type = 'Pacs_003'"

	fmt.Println("\n\n", query)

	rows, err := db.Query(query)
	if err != nil {
		log.Print(err.Error())
	}
	for rows.Next() {
		var transactions int64

		rows.Scan(&transactions)
		day1_MP_WAIT = transactions

	}

	message := fmt.Sprintf("Transactions waiting for EDO Postin \nMP_WAIT - %s : %d", day1Date, day1_MP_WAIT)
	if (day1_WAITSCHEDSUBBATCH + day0_SCHEDULE) != day1_MP_WAIT {
		message = message + fmt.Sprintf("\nRemedial action needed: Expected %d  \n ", (day1_WAITSCHEDSUBBATCH+day0_SCHEDULE))
		alerting.Callout(message)
	}

	fmt.Println(message)
	alerting.Info(message)

}

func getSCHEDULEcount() {
	db = database.NewConnection()
	defer db.Close()
	logNow()

	query := "SELECT count(*) transactions FROM gpp_sp.minf WHERE p_msg_sts = 'SCHEDULE' AND p_dbt_vd = '" + day0Date + "'  and p_msg_type = 'Pacs_003'"

	fmt.Println(" \n\n", query)

	rows, err := db.Query(query)
	if err != nil {
		log.Print(err.Error())
	}
	for rows.Next() {
		var transactions int64

		rows.Scan(&transactions)

		day0_SCHEDULE = transactions

	}

	message := fmt.Sprintf("Tracking Transactions \nSCHEDULE - %s : %d ", day0Date, day0_SCHEDULE)

	fmt.Println(message)
	alerting.Info(message)

}

func edoFilesOutGoing() {
	logNow()
	command := "find /cdwasha/connectdirect/outgoing/EDO_DirectDebitRequest -type f -cmin -60 -name 'EDO_POST*' -exec wc -l {} \\; "

	fmt.Println("edoFilesOutGoing\n", command)
	message := ""
	output, err := remote.RemoteSsh(command)
	if err != nil {

		fmt.Println("error:", err.Error())
		if err.Error() == "Process exited with status 1" {
			message = "Outgoing edo file check failed "
			fmt.Println(message)
		}
		alerting.Callout(message)
		fmt.Println(message)
		return
	}

	fmt.Println(output)
	outputSlice := strings.Split(output, " ")
	linecount, _ := strconv.Atoi(outputSlice[0])
	if linecount == 0 {
		message = "Tracking file not found in /cdwasha/connectdirect/outgoing/EDO_DirectDebitRequest "
		alerting.Callout(message)
		return
	}
	records := linecount - 2
	day1_edoPosting = int64(records)
	message = fmt.Sprintf("Tracking file recived from EDO contains %d records \n", day1_edoPosting)
	if day1_edoPosting != day1_MP_WAIT {
		message += fmt.Sprintf("\nExpected %d ", day1_MP_WAIT)
		alerting.Callout(message)
	} else {
		alerting.Info(message)
		fmt.Println(message)
		fmt.Println("\nedoFilesOutGoing --end\n")
	}

}

func edoFilesOutGoingArchived() {
	logNow()
	command := "find /cdwasha/connectdirect/outgoing/EDO_DirectDebitRequest -type f -cmin -60 -name 'EDO_POST*' -exec wc -l {} \\; "

	fmt.Println("edoFilesOutGoing\n", command)
	message := ""
	output, err := remote.RemoteSsh(command)
	if err != nil {

		fmt.Println("error:", err.Error())
		if err.Error() == "Process exited with status 1" {
			message = "Outgoing edo file check failed "
			fmt.Println(message)
		}
		alerting.Callout(message)
		fmt.Println(message)
		return
	}

	fmt.Println(output)
	outputSlice := strings.Split(output, " ")
	linecount, _ := strconv.Atoi(outputSlice[0])
	if linecount == 0 {
		message = "EDO_POSTING file not found in /cdwasha/connectdirect/outgoing/EDO_DirectDebitRequest/archive "
		alerting.Callout(message)
		return
	}
	records := linecount - 2

	day1_edoPostingArchived = int64(records)
	message = fmt.Sprintf("Archived EDO_POSTING file contains %d records \n", day1_edoPostingArchived)
	if day1_edoPosting != day1_MP_WAIT {
		message += fmt.Sprintf("\nExpected %d ", day1_MP_WAIT)
		alerting.Callout(message)
	} else {
		alerting.Info(message)
		fmt.Println(message)
		fmt.Println("\nedoFilesOutGoing --end\n")
	}

}

func edoResponseLEG() {
	logNow()
	fmt.Println("EdoResponseLEG\n")
	defaultFormat := "2006-01-02"

	today := time.Now().Format(defaultFormat)

	command := "wc -l /cdwasha/connectdirect/incoming/EDO_DirectDebitResponse/archive/" + today + "*ACDEBIT.RESPONSE.LEG.*"
	message := ""
	fmt.Println(command)
	remote.RemoteSsh(command)
	output, err := remote.RemoteSsh(command)
	///
	if err != nil {

		fmt.Println("error:", err.Error())
		if err.Error() == "Process exited with status 1" {
			message = "EDO ResponseLEG file check failed "
			fmt.Println(message)
		}
		alerting.Callout(message)
		fmt.Println(message)
		return
	}

	fmt.Println(output)
	outputSlice := strings.Split(output, " ")
	linecount, _ := strconv.Atoi(outputSlice[0])
	if linecount == 0 {
		message = "EDO ResponseLEG file not found in /cdwasha/connectdirect/incoming/EDO_DirectDebitResponse/archive/"
		alerting.Callout(message)
		return
	}
	records := linecount - 2

	day1_legacyResponse = int64(records)
	message = fmt.Sprintf("Leagacy Resaponse file contains %d records \n", day1_legacyResponse)

	alerting.Info(message)
	fmt.Println(message)

	fmt.Println("EdoResponseLEG --end\n")

}

func edoResponseSAP() {
	logNow()
	defaultFormat := "2006-01-02"

	today := time.Now().Format(defaultFormat)

	command := "wc -l /cdwasha/connectdirect/incoming/EDO_DirectDebitResponse/archive/" + today + "*ACDEBIT.RESPONSE.SAP.*"
	fmt.Println("EdoResponseSAP\n", command)
	var message string
	output, err := remote.RemoteSsh(command)
	///
	if err != nil {

		fmt.Println("error:", err.Error())
		if err.Error() == "Process exited with status 1" {
			message = "EDO Response SAP file check failed "
			fmt.Println(message)
		}
		alerting.Callout(message)
		fmt.Println(message)
		return
	}

	fmt.Println(output)
	outputSlice := strings.Split(output, " ")
	linecount, _ := strconv.Atoi(outputSlice[0])
	if linecount == 0 {
		message = "EDO Response SAP file not found in /cdwasha/connectdirect/incoming/EDO_DirectDebitResponse/archive/"
		alerting.Callout(message)
		return
	}
	records := linecount - 2

	day1_sapResponse = int64(records)
	message = fmt.Sprintf("SAP Response file contains %d records \n", day1_sapResponse)

	alerting.Info(message)
	fmt.Println(message)


	fmt.Println("EdoResponseSAP --end\n")
}

func allStatuses() {
	db = database.NewConnection()
	defer db.Close()
	logNow()

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

	//fmt.Println(" \n\n", query)
	fmt.Println("\n")

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

		fmt.Printf("%-10s  -  %-20s  -  %-10s  -  %5d \n", datum[:10], status, msg_type, transactions)
	}

}

func sendtelegram() {}

func callout() {}
