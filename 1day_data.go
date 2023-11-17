// 差价天数据
package main

import (
	"RTJws/tool"
	"fmt"
	client "github.com/influxdata/influxdb1-client/v2"
	"log"
	"strconv"
	"time"
)

func main() {
	database := "jdm"
	cli := tool.ConnInflux("http://120.24.148.136:8086")
	// 获取今天的开始时间和结束时间
	// 获取当前时间
	now := time.Now()
	// 获取当天的开始时间（零点整）
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	startOfDayStr := startOfDay.Format("2006-01-02T15:04:05Z")
	// 获取当天的结束时间（23:59:59）
	endOfDay := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, now.Location())
	endOfDayStr := endOfDay.Format("2006-01-02T15:04:05Z")
	//fmt.Printf("%T", startOfDayStr)
	//fmt.Println(endOfDayStr)
	//startOfDayStr = "2023-06-27T00:00:00Z"
	//endOfDayStr = "2023-06-27T23:59:59Z"
	// sql
	maxSql := fmt.Sprintf(`SELECT max(high) FROM "1m_SHAU_SQAU_CJ" where time > '%s' and time < '%s'`, startOfDayStr, endOfDayStr)
	minSql := fmt.Sprintf(`SELECT min(low) FROM "1m_SHAU_SQAU_CJ" where time > '%s' and time < '%s'`, startOfDayStr, endOfDayStr)
	openSql := fmt.Sprintf(`SELECT open FROM "1m_SHAU_SQAU_CJ" where time > '%s' and time < '%s' order by asc limit 1`, startOfDayStr, endOfDayStr)
	closeSql := fmt.Sprintf(`SELECT close FROM "1m_SHAU_SQAU_CJ" where time > '%s' and time < '%s'  order by desc limit 1`, startOfDayStr, endOfDayStr)
	buy_sellSql := fmt.Sprintf(`SELECT buy,sell FROM "1m_SHAU_SQAU_CJ" where time > '%s' and time < '%s'  order by desc limit 1`, startOfDayStr, endOfDayStr)
	fields := map[string]interface{}{
		"volume": 0,
	}
	max := client.NewQuery(maxSql, database, "")
	// Execute query
	if response, err := cli.Query(max); err == nil && response.Error() == nil {
		if len(response.Results[0].Series) == 0 {
			log.Println("今天无数据！")
			return
		}
		fields["high"] = response.Results[0].Series[0].Values[0][1]
		fmt.Println(response.Results[0].Series[0].Values[0][1])
	} else {
		log.Fatal(err)
	}
	buy_sell := client.NewQuery(buy_sellSql, database, "")
	// Execute query
	if response, err := cli.Query(buy_sell); err == nil && response.Error() == nil {
		fields["buy"] = response.Results[0].Series[0].Values[0][1]
		fields["sell"] = response.Results[0].Series[0].Values[0][2]
		fmt.Println(response.Results[0].Series[0].Values[0][1])
		fmt.Println(response.Results[0].Series[0].Values[0][2])
	}
	min := client.NewQuery(minSql, database, "")
	// Execute query
	if response, err := cli.Query(min); err == nil && response.Error() == nil {
		fields["low"] = response.Results[0].Series[0].Values[0][1]
		fmt.Println(response.Results[0].Series[0].Values[0][1])
	}
	open := client.NewQuery(openSql, database, "")
	// Execute query
	if response, err := cli.Query(open); err == nil && response.Error() == nil {
		fields["open"] = response.Results[0].Series[0].Values[0][1]
		fmt.Println("open：", response.Results[0].Series[0].Values[0][1])
	}
	close := client.NewQuery(closeSql, database, "")
	// Execute query
	if response, err := cli.Query(close); err == nil && response.Error() == nil {
		fields["close"] = response.Results[0].Series[0].Values[0][1]
		fmt.Println(response.Results[0].Series[0].Values[0][1])
	}
	bp, err := client.NewBatchPoints(client.BatchPointsConfig{
		Database:  database,
		Precision: "s", //精度，默认ns
	})
	// day
	startOfDayTimestamp := startOfDay.Unix()
	// 获取本月的开始时间
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	// 计算距离上一周一的天数
	daysSinceMonday := int(now.Weekday()) - 1
	if daysSinceMonday < 0 {
		daysSinceMonday = 6
	}
	// 计算本周一的日期
	monday := now.AddDate(0, 0, -daysSinceMonday)
	// 设置本周一的时间为凌晨 00:00:00
	weekStart := time.Date(monday.Year(), monday.Month(), monday.Day(), 0, 0, 0, 0, monday.Location())
	// 获取开始时间的时间戳（Unix时间戳，单位秒）
	startOfMonthTimeStamp := startOfMonth.Unix()
	startOfWeekTimeStamp := weekStart.Unix()
	startOfDayTimestampStr := strconv.FormatInt(startOfDayTimestamp, 10)
	startOfMonthTimestampStr := strconv.FormatInt(startOfMonthTimeStamp, 10)
	startOfWeekTimestampStr := strconv.FormatInt(startOfWeekTimeStamp, 10)
	tags := map[string]string{
		"day":   startOfDayTimestampStr,
		"month": startOfMonthTimestampStr,
		"week":  startOfWeekTimestampStr,
	}
	pt, err := client.NewPoint("1d_SHAU_SQAU_CJ", tags, fields, time.Now())
	if err != nil {
		log.Fatal(err)
	}
	bp.AddPoint(pt)
	err = cli.Write(bp)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("insert success")
}
