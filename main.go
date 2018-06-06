package main

import (
	"database/sql"
	"fmt"
	"github.com/igknot/gppStandby/database"
	"github.com/jasonlvhit/gocron"
	"log"

	"time"
	"github.com/igknot/gppStandby/alerting"
	"github.com/igknot/gppStandby/remote"
	"strings"
	"strconv"
)

var db *sql.DB
var zaDate, xxxDate, edoFileName , day0Date, day1Date string
var  newTrans, trackedTrans, scheduledTrans, newAndtracked, trackedFromEDO int64

var day1_WAITSCHEDSUBBATCH , day0_SCHEDULE , day1_MP_WAIT ,day0_NightTrackingFile ,edoPosting ,edoPostingArchive ,sapResponse , legacyResponse int64


func main() {
	defaultFormat := "02/Jan/2006"
	day0Date = time.Now().AddDate(0, 0, 1).Format(defaultFormat)
	day1Date = time.Now().Format(defaultFormat)

	edoTrackingFileSAPLEG()
	//allStatuses()
	//
	//getRolloverdate("ZA1")
	//getRolloverdate("***")
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

	//2 global date rollover 00:22
	scheduler.Every(1).Day().At("00:22").Do(getRolloverdate, "ZA1")

	//3 new transactions
	scheduler.Every(1).Day().At("23:55").Do(getWAITSCHEDSUBBATCHcount)

	//4 Tracking  transactions
	scheduler.Every(1).Day().At("23:50").Do(trackingTransactions)

	//5 Scheduled 00:08
	// 00:08 (00:01) track should be 0 and scheduled should be value of old_tracking after
	//scheduled and wait scheduled  will add up to  newandtracked
	scheduler.Every(1).Day().At("00:08").Do(ScheduledTransactions)

	//6 new and tracked 00:37
	scheduler.Every(1).Day().At("00:37").Do(newAndTracked)

	//7 Check edo files 00:57 + 2minutes for safety
	scheduler.Every(1).Day().At("00:57").Do(edoFilesOutGoing)

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

	logNow()

	//23:31 and 00:21
	getRolloverdate("ZA1")
	getRolloverdate("***")
	getWAITSCHEDSUBBATCHcount()
	trackingTransactions()
	ScheduledTransactions()
	newAndTracked()
	edoFilesOutGoing()        //00:57
	edoFileArchived() //01:01
	edoResponseSAP()  //anytime before 01:30 or 02:30 send mail to rcop if they are not there
	edoResponseLEG()

	fmt.Println("ZADATE: ", zaDate, "\nGLOBALDATE ", xxxDate, "\nNEW:", newTrans, "\nTracked", trackedTrans, "\nScheduled: ", scheduledTrans,
		"\nNewAndTrackedc", newAndtracked)
}
func reset() {
	logNow()
	zaDate = ""
	xxxDate = ""
	newTrans = 0
	trackedTrans = 0
	scheduledTrans = 0
	newAndtracked = 0

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
			zaDate = bsnessdate[:10]
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
	//defaultFormat := "02/Jan/2006"
	//defaultFormat := "02/Jan/2006"
	//
	//tomorrow := time.Now().AddDate(0, 0, 1).Format(defaultFormat)

	query := "SELECT count(*) transactions FROM gpp_sp.minf WHERE p_msg_sts = 'WAITSCHEDSUBBATCH' AND p_dbt_vd = '" + day1Date + "'  and p_msg_type = 'Pacs_003'"

	fmt.Println("\n\n", query)

	rows, err := db.Query(query)
	if err != nil {
		log.Print(err.Error())
	}
	for rows.Next() {
		var transactions int64

		rows.Scan(&transactions)

		message := fmt.Sprintf("WAITSCHEDSUBBATCH - %s : %d ",day1Date, transactions)
		newTrans = transactions
		fmt.Println(message)
		alerting.Info(message)
	}

}
func edoTrackingFileSAPLEG(){
	// after 00:01
	// this should check whether /cdwasha/connectdirect/incoming/EDO_DirectDebitResponse/*SAP.LEG file exists
	// If the file has not been recieved
	//			Change: operations>interfaces EDO_POSTING_REQ to inactive  > save
	//					operations>apply changes> interface
	//			After file has been recieved
	//					operations> Tasks > New day Activities  > generate posting req for EDO > EDO_NEW > execute
	logNow()
	fmt.Println("edoTrackingFileSAPLEG\n")
	defaultFormat := "2006-01-02"

	today := time.Now().Format(defaultFormat)

	command := "wc -l /cdwasha/connectdirect/incoming/EDO_DirectDebitResponse/"+today+"*ACDEBIT.RESPONSE.LEG.SAP*"



	fmt.Println( command)
	remote.RemoteSsh(command)
	output, err := remote.RemoteSsh(command)
	if err != nil {
		fmt.Println("error-recieved\noutput:",output)
		fmt.Println("error:",err.Error())
		return
	}
	fmt.Println(output)
	outputSlice := strings.Split(output, " ")
	linecount,_ := strconv.Atoi(outputSlice[0])
	records := linecount - 2
	trackedFromEDO = int64(records)
	fmt.Printf("Tracking file recived from EDO contains %d records \n", trackedFromEDO)
	fmt.Println("\nedoTrackingFileSAPLEG --end\n")



}
func trackingTransactions() {

	//

	db = database.NewConnection()
	defer db.Close()
	logNow()
	defaultFormat := "02/Jan/2006"
	today := time.Now().Format(defaultFormat)
	//tomorrow := time.Now().AddDate(0, 0, 1).Format(defaultFormat)

	query := "SELECT count(*) transactions FROM gpp_sp.minf WHERE p_msg_sts = 'MP_WAIT' AND p_dbt_vd = '" + today + "'  and p_msg_type = 'Pacs_003'"

	fmt.Println("\n\n", query)

	rows, err := db.Query(query)
	if err != nil {
		log.Print(err.Error())
	}
	for rows.Next() {
		var transactions int64

		rows.Scan(&transactions)

		message := fmt.Sprintf("Tracking transactions(MP_WAIT) %d", transactions)
		trackedTrans = transactions
		fmt.Println(message)
		alerting.Info(message)
	}

}

func ScheduledTransactions() {
	db = database.NewConnection()
	defer db.Close()
	logNow()
	defaultFormat := "02/Jan/2006"
	today := time.Now().Format(defaultFormat)

	query := "SELECT count(*) transactions FROM gpp_sp.minf WHERE p_msg_sts LIKE 'SCH%' AND p_dbt_vd = '" + today + "'  and p_msg_type = 'Pacs_003'"

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
	message := fmt.Sprintf("Tracked : %d \nScheduled: %d  \n", trackedTrans, scheduledTrans)

	if (trackedTrans) != scheduledTrans {
		message = message + fmt.Sprintf("Alert  - (trackedTrans) != scheduledTrans ")
	} else {
		message = message +" GOOD "
	}

}

func newAndTracked() {
	db = database.NewConnection()
	defer db.Close()
	logNow()

	defaultFormat := "02/Jan/2006"
	today := time.Now().Format(defaultFormat)

	query := "SELECT count(*) transactions FROM gpp_sp.minf WHERE p_msg_sts LIKE 'MP_WAIT%' AND p_dbt_vd = '" + today + "'  and p_msg_type = 'Pacs_003' "

	fmt.Println(" \n\n", query)

	rows, err := db.Query(query)
	if err != nil {
		log.Print(err.Error())
	}
	var message string
	for rows.Next() {
		var transactions int64

		rows.Scan(&transactions)

		message = fmt.Sprintf("New and tracked(MP_WAIT)", transactions)
		newAndtracked = transactions
	}
	if newAndtracked != (scheduledTrans + newTrans) {
		message += "\nA<b>ALERT:</b>New MP_WAIT does not add up scheduled + new "
	}
	alerting.Info(message)
}
func edoFilesOutGoing() {
	logNow()
	command := "find /cdwasha/connectdirect/outgoing/EDO_DirectDebitRequest -type f -cmin -60 -name 'EDO_POST*' -exec wc -l {} \\; "
	// 00:57

	fmt.Println("EdoFiles\n", command)

	output, err := remote.RemoteSsh(command)
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
	output, err := remote.RemoteSsh(command)
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
	remote.RemoteSsh(command)
	output, err := remote.RemoteSsh(command)
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
	output, err := remote.RemoteSsh(command)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	fmt.Println(output)
	fmt.Println("EdoResponseSAP --end\n")
}
func allStatuses() {
	db = database.NewConnection()
	defer db.Close()
	logNow()

	days := "'"  + day0Date + "','"+ day1Date +"'"

	query := `SELECT
	    p_dbt_vd datum,
		p_msg_sts status,
		p_msg_type msg_type,
		COUNT(*) transactions
	FROM
	gpp_sp.minf
	WHERE
	P_DBT_VD in (` + days +`)
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

		rows.Scan(&datum , &status, &msg_type,&transactions)

		fmt.Printf("%-10s  -  %-20s  -  %-10s  -  %5d \n", datum[:10] , status, msg_type,transactions)
	}

}


func sendtelegram() {}

func callout() {}
