// 获取上期所的数据并从redis中拿出SHAU的数据进行求差，得到套利数据 SHAU_SQAU_CJ
// 从redis中拿出数据 AUTD_SHAU_CJ
package main

import (
	"RTJws/tool"
	"encoding/json"
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/gorilla/websocket"
	client "github.com/influxdata/influxdb1-client/v2"
	"github.com/shopspring/decimal"
	"log"
	"net"
	"os"
	"time"
)

var configuration *tool.Configurations

func main() {
	configuration = new(tool.Configurations)
	// 获取命令行参数
	args := os.Args
	path := "./config"
	if len(args) > 1 && len(args) < 3 {
		path = args[1]
	}
	fmt.Println("path：", path)
	tool.InitConfig(path, configuration)
	fmt.Println("开始打印配置参数")
	fmt.Println(configuration.Ws.Server_address, configuration.Redis.Addr, configuration.Redis.Pwd, configuration.Redis.Db)
	fmt.Println(configuration.Udpserver.Address, configuration.Influxdb.Address)
	// redis
	rdb := tool.ConnRedis(configuration.Redis.Addr, configuration.Redis.Pwd, configuration.Redis.Db)
	defer rdb.Close()
	// WebSocket 服务器的地址
	serverAddr := configuration.Ws.Server_address

	// 创建 WebSocket 连接
	conn, _, err := websocket.DefaultDialer.Dial(serverAddr, nil)
	if err != nil {
		log.Fatal("Error connecting to WebSocket server:", err)
	}
	defer conn.Close()
	// 定义一个udp地址
	addr, err := net.ResolveUDPAddr("udp", configuration.Udpserver.Address)
	if err != nil {
		log.Fatal(err)
	}
	// 创建一个udp连接
	udpConn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		log.Fatal(err)
	}
	// 获取环境变量中的env
	cli := tool.ConnInflux(configuration.Influxdb.Address)

	// 启动一个单独的 goroutine 来接收和处理服务器发送的消息
	go receiveMessagesSendUDP(conn, rdb, udpConn, cli)
	go countTD_SH(rdb, udpConn, cli)
	// 在这里可以继续发送和接收消息，进行其他操作
	// 暂停程序，防止立即退出
	fmt.Println("Press Enter to exit...")
	fmt.Scanln()
}

func getDataFromRedis(key string, rdb *redis.Client) (tool.RedisData, error) {
	// 添加key为空不查询
	if key == "" {
		return tool.RedisData{}, nil
	}
	val, err := rdb.Get(rdb.Context(), key).Result()
	if err != nil {
		return tool.RedisData{}, err
	}
	var redisData tool.RedisData
	if err = json.Unmarshal([]byte(val), &redisData); err != nil {
		return tool.RedisData{}, err
	}
	return redisData, nil
}

func processRedisData(key1, key2 string, rdb *redis.Client, processFunc func(redisData1, redisData2 tool.RedisData)) {
	redisData1, err := getDataFromRedis(key1, rdb)
	redisData2, err := getDataFromRedis(key2, rdb)
	if err != nil {
		log.Println("Error getting value:", err)
		return
	}
	processFunc(redisData1, redisData2)
}

func countTD_SH(rdb *redis.Client, udpConn *net.UDPConn, cli client.Client) {
	var data tool.DataSource
	data.Code = "AUTD_SHAU_CJ"
	dataMap := make(map[int64][]float64)
	defer cli.Close()
	for {
		processRedisData("last:AUTD", "last:SHAU", rdb, func(redisData1, redisData2 tool.RedisData) {
			sell := redisData1.Sell - redisData1.Sell
			sell_res := tool.Decimal(sell)
			timestamp := time.Now().Truncate(time.Minute)
			prev := timestamp.Add(-time.Minute)
			// 上一分钟时间戳
			prevUnix := prev.Unix()
			// 当前分钟时间戳
			unix := timestamp.Unix()
			// 判断上一个map的数据是否存在
			if _, ok := dataMap[prevUnix]; ok {
				delete(dataMap, prevUnix)
			}
			// 每分钟的数据实时组合推送到实时k线
			dataMap[unix] = append(dataMap[unix], sell_res)
			high, _ := tool.MaxNum(dataMap[unix])
			low, _ := tool.MinNum(dataMap[unix])
			data.Open = dataMap[unix][0]
			data.LastClose = dataMap[unix][len(dataMap[unix])-1]
			data.Sell = dataMap[unix][len(dataMap[unix])-1]
			x := decimal.NewFromFloat(data.Sell)
			y := decimal.NewFromFloat(0.02)
			z := x.Sub(y).Round(2)
			f, _ := z.Float64()
			data.Buy = f
			data.High = high
			data.Low = low
			data.QuoteTime = unix
			data.Last = data.Sell
			tool.WritesPoints(cli, data)
			str, _ := json.Marshal(data)
			fmt.Println("unix_str：", string(str))
			_, err := udpConn.Write(str)
			if err != nil {
				fmt.Println(err)
			}
		})
	}
}

// 接收和处理服务器发送的消息
func receiveMessagesSendUDP(conn *websocket.Conn, rdb *redis.Client, udpConn *net.UDPConn, cli client.Client) {
	var data tool.DataSource
	data.Code = "SHAU_SQAU_CJ"
	dataMap := make(map[int64][]float64)
	defer cli.Close()
	for {
		// 读取服务器发送的消息
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Fatal("Error reading message from WebSocket server:", err)
			return
		}
		var requestdata tool.RequestData
		if err = json.Unmarshal(msg, &requestdata); err != nil {
			log.Println("err：", err)
		}
		if requestdata.Group == "水贝价VS上期" {
			switch requestdata.ID {
			case "g004-1":
				processRedisData("last:SHAU", "", rdb, func(redisData1, redisData2 tool.RedisData) {
					sell := requestdata.RightValue - redisData1.Sell
					sell_res := tool.Decimal(sell)
					timestamp := time.Now().Truncate(time.Minute)
					prev := timestamp.Add(-time.Minute)
					// 上一分钟时间戳
					prevUnix := prev.Unix()
					// 当前分钟时间戳
					unix := timestamp.Unix()
					// 判断上一个map的数据是否存在
					if _, ok := dataMap[prevUnix]; ok {
						delete(dataMap, prevUnix)
					}
					// 每分钟的数据实时组合推送到实时k线
					dataMap[unix] = append(dataMap[unix], sell_res)
					high, _ := tool.MaxNum(dataMap[unix])
					low, _ := tool.MinNum(dataMap[unix])
					data.Open = dataMap[unix][0]
					data.LastClose = dataMap[unix][len(dataMap[unix])-1]
					data.Sell = dataMap[unix][len(dataMap[unix])-1]
					x := decimal.NewFromFloat(data.Sell)
					y := decimal.NewFromFloat(0.02)
					z := x.Sub(y).Round(2)
					f, _ := z.Float64()
					data.Buy = f
					data.High = high
					data.Low = low
					data.QuoteTime = unix
					data.Last = data.Sell
					tool.WritesPoints(cli, data)
					str, _ := json.Marshal(data)
					fmt.Println("unix_str：", string(str))
					_, err = udpConn.Write(str)
					if err != nil {
						fmt.Println(err)
					}
				})
			}
		}
	}
}
