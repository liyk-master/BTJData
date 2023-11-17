// 百通金数据获取
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"
)

type Quote struct {
	State string `json:"state"`
	Items []any  `json:"items"`
}
type UdpData struct {
	Code      string  `json:"Code"`
	LastClose int     `json:"LastClose"`
	QuoteTime int64   `json:"QuoteTime"`
	Volume    int     `json:"Volume"`
	Buy       float64 `json:"Buy"`
	High      float64 `json:"High"`
	Low       float64 `json:"Low"`
	Open      float64 `json:"Open"`
	Last      float64 `json:"Last"`
	Sell      float64 `json:"Sell"`
}

func getHttpData(codes map[string]string, conn *net.UDPConn) {
	// 创建一个HTTP客户端
	client := &http.Client{}

	// 创建一个GET请求
	req, err := http.NewRequest("GET", "http://www.btj9999.com/get_price.php", nil)
	if err != nil {
		log.Fatal(err)
	}

	// 发送请求并获取响应
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}

	// 关闭响应的正文
	defer resp.Body.Close()

	// 打印响应的状态码
	if resp.Status != "200 OK" {
		fmt.Println("err", "数据接口异常！")
		return
	}

	// 读取并打印响应的正文
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	var respStu Quote
	if err = json.Unmarshal(body, &respStu); err != nil {
		fmt.Println("err：", err)
		return
	}

	var udpdata UdpData
	for _, v := range respStu.Items {
		if m, ok := v.(map[string]any); ok {
			value, ok := codes[m["code"].(string)]
			if !ok {
				return
			}
			udpdata.Code = value
			// 使用strconv包的ParseFloat函数将字符串类型的值转换为一个浮点数类型的值
			//fmt.Println(fmt.Sprintf("%T\n%T\n%T\n%T", m["askprice"], m["bidprice"], m["high"], m["low"]))
			udpdata.Sell = m["askprice"].(float64)
			udpdata.Buy = m["bidprice"].(float64)
			udpdata.High, _ = strconv.ParseFloat(m["high"].(string), 64)
			udpdata.Low, _ = strconv.ParseFloat(m["low"].(string), 64)
			udpdata.QuoteTime = time.Now().Unix()
		}
		// udp推送
		// 将udpdata结构体转换为json格式的字节切片
		data, err := json.Marshal(udpdata)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println("data：", string(data))
		// 使用Write函数来发送数据
		_, err = conn.Write(data)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func main() {
	var codes map[string]string
	codes = make(map[string]string)
	codes["JZJ_au"] = "BTJ_AU"
	codes["JZJ_ag"] = "BTJ_AG"
	codes["JZJ_pt"] = "BTJ_AP"
	codes["JZJ_pd"] = "BTJ_PD"
	codes["Au99.99"] = "RT_AU9999"
	codes["Au(T+D)"] = "RT_AUTD"
	codes["Ag(T+D)"] = "RT_AGTD"
	codes["Pt99.95"] = "RT_AP9995"
	codes["GLNC"] = "RT_GC"
	codes["PLNC"] = "RT_PL"
	codes["PANC"] = "RT_PA"
	codes["SLNC"] = "RT_SI"
	codes["XAU"] = "RT_XAU"
	codes["XAG"] = "RT_XAG"
	codes["XAP"] = "RT_XAP"
	codes["XPD"] = "RT_XPD"
	codes["USDCNH"] = "RT_USDCNH"
	// 获取命令行参数
	args := os.Args
	address := "172.26.179.28:7158"
	if len(args) > 1 && len(args) < 3 {
		address = args[1]
	}
	fmt.Println("address：", address)
	// 定义一个udp地址
	addr, err := net.ResolveUDPAddr("udp", address)
	if err != nil {
		log.Fatal(err)
	}
	// 创建一个udp连接
	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	//getHttpData(codes, conn, addr)
	// 创建一个秒定时器
	ticker := time.NewTicker(1 * time.Second)

	// 在每一秒打印一个数字
	for range ticker.C {
		getHttpData(codes, conn)
	}
}
