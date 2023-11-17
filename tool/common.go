package tool

import (
	"fmt"
	"github.com/go-redis/redis/v8"
	client "github.com/influxdata/influxdb1-client/v2"
	"github.com/spf13/viper"
	"log"
	"strconv"
	"time"
)

type RequestData struct {
	ID          string  `json:"ID"`
	LeftValue   float64 `json:"LeftValue"`
	RightValue  float64 `json:"RightValue"`
	DiffValue   float64 `json:"DiffValue"`
	Time        string  `json:"Time"`
	Name        string  `json:"Name"`
	Group       string  `json:"Group"`
	Decimal     int     `json:"Decimal "`
	IsShowValue bool    `json:"IsShowValue "`
}
type RedisData struct {
	C    string  `json:"C"`
	T    int     `json:"T"`
	P    float64 `json:"P"`
	L    float64 `json:"L"`
	H    float64 `json:"H"`
	O    float64 `json:"O"`
	V    int     `json:"V"`
	Buy  float64 `json:"Buy"`
	Sell float64 `json:"Sell"`
	Lc   float64 `json:"LC"`
	Zd   float64 `json:"ZD"`
	Zdf  float64 `json:"ZDF"`
}

type DataSource struct {
	Code      string  `json:"Code"`
	Volume    int     `json:"Volume"`
	QuoteTime int64   `json:"QuoteTime"`
	Last      float64 `json:"Last"`
	Open      float64 `json:"Open"`
	High      float64 `json:"High"`
	Low       float64 `json:"Low"`
	LastClose float64 `json:"LastClose"`
	Buy       float64 `json:"Buy"`
	Sell      float64 `json:"Sell"`
}

type Configurations struct {
	Ws        WsConfigurations
	Redis     RedisConfigurations
	Udpserver UdpserverConfigurations
	Influxdb  InfluxdbConfigurations
}

type WsConfigurations struct {
	Server_address string
}

type RedisConfigurations struct {
	Addr string
	Pwd  string
	Db   int
}
type UdpserverConfigurations struct {
	Address string
}
type InfluxdbConfigurations struct {
	Address string
}

func InitConfig(path string, configuration *Configurations) {
	viper.SetConfigName("config")
	viper.AddConfigPath(path)
	viper.AutomaticEnv()
	viper.SetConfigType("yml")
	if err := viper.ReadInConfig(); err != nil {
		fmt.Printf("Error reading config file, %s", err)
	}
	err := viper.Unmarshal(&configuration)
	if err != nil {
		fmt.Printf("Unable to decode into struct, %v", err)
	}
}

func Decimal(value float64) float64 {
	value, _ = strconv.ParseFloat(fmt.Sprintf("%.2f", value), 64)
	return value
}

// influxdb demo
func ConnInflux(addr string) client.Client {
	cli, err := client.NewHTTPClient(client.HTTPConfig{
		Addr: addr,
		//Addr: "http://120.24.148.136:8086",
		//Addr: "http://192.168.1.16:8086",
		//Username: "admin",
		//Password: "",
	})
	if err != nil {
		log.Fatal(err)
	}
	return cli
}

func ConnRedis(addr, passwd string, db int) *redis.Client {
	// redis
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: passwd,
		DB:       db,
	})

	_, err := rdb.Ping(rdb.Context()).Result()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("成功连接redis")
	return rdb
}

func MaxNum(arr []float64) (max float64, maxIndex int) {
	max = arr[0] //假设数组的第一位为最大值
	//常规循环，找出最大值
	for i := 0; i < len(arr); i++ {
		if max < arr[i] {
			max = arr[i]
			maxIndex = i
		}
	}
	return max, maxIndex
}

func MinNum(arr []float64) (min float64, minIndex int) {
	min = arr[0] //假设数组的第一位为最小值
	//for-range循环方式，找出最小值
	for index, val := range arr {
		if min > val {
			min = val
			minIndex = index
		}
	}
	return min, minIndex
}

// insert
func WritesPoints(cli client.Client, data DataSource) {
	bp, err := client.NewBatchPoints(client.BatchPointsConfig{
		Database:  "jdm",
		Precision: "s", //精度，默认ns
	})
	if err != nil {
		log.Fatal(err)
	}
	now := time.Now()
	// day
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	startOfDayTimestamp := startOfDay.Unix()
	// 计算距离上一周一的天数
	daysSinceMonday := int(now.Weekday()) - 1
	if daysSinceMonday < 0 {
		daysSinceMonday = 6
	}
	// 计算本周一的日期
	monday := now.AddDate(0, 0, -daysSinceMonday)
	// 设置本周一的时间为凌晨 00:00:00
	weekStart := time.Date(monday.Year(), monday.Month(), monday.Day(), 0, 0, 0, 0, monday.Location())
	startOfWeekTimestamp := weekStart.Unix()
	year, month, _ := now.Date()
	// month
	startOfMonth := time.Date(year, month, 1, 0, 0, 0, 0, now.Location())
	startOfMonthTimestamp := startOfMonth.Unix()
	startOfDayTimestampStr := strconv.FormatInt(startOfDayTimestamp, 10)
	startOfMonthTimestampStr := strconv.FormatInt(startOfMonthTimestamp, 10)
	startOfWeekTimestampStr := strconv.FormatInt(startOfWeekTimestamp, 10)
	tags := map[string]string{
		"day":   startOfDayTimestampStr,
		"month": startOfMonthTimestampStr,
		"week":  startOfWeekTimestampStr,
	}
	fields := map[string]interface{}{
		"buy":    data.Buy,
		"close":  data.LastClose,
		"high":   data.High,
		"low":    data.Low,
		"open":   data.Open,
		"sell":   data.Sell,
		"volume": data.Volume,
	}
	pt, err := client.NewPoint(fmt.Sprintf("%s_%s", "1m", data.Code), tags, fields, time.Now().Truncate(time.Minute))
	if err != nil {
		log.Fatal(err)
	}
	bp.AddPoint(pt)
	err = cli.Write(bp)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("insert success")
}
