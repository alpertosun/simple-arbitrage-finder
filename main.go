package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"
)

type ExcList struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Exc struct {
	Name                        string      `json:"name"`
	YearEstablished             int         `json:"year_established"`
	Country                     interface{} `json:"country"`
	Description                 string      `json:"description"`
	TrustScore                  int         `json:"trust_score"`
	TrustScoreRank              int         `json:"trust_score_rank"`
	TradeVolume24HBtc           float64     `json:"trade_volume_24h_btc"`
	TradeVolume24HBtcNormalized float64     `json:"trade_volume_24h_btc_normalized"`
	Tickers                     []struct {
		Base          string  `json:"base"`
		Target        string  `json:"target"`
		Last          float64 `json:"last"`
		Volume        float64 `json:"volume"`
		ConvertedLast struct {
			Btc float64 `json:"btc"`
			Eth float64 `json:"eth"`
			Usd float64 `json:"usd"`
		} `json:"converted_last"`
		ConvertedVolume struct {
			Btc float64 `json:"btc"`
			Eth int     `json:"eth"`
			Usd int     `json:"usd"`
		} `json:"converted_volume"`
		TrustScore string `json:"trust_score"`
	} `json:"tickers"`
	StatusUpdates []interface{} `json:"status_updates"`
}

var excList []ExcList
var exc []Exc

const telegramBotApi = "telegram_bot_api_token"
const telegramChannel = "@telegram_public_channel_name"
const minVolume = 30000
const pair = "USDT"
const ratio = 1.1

var enabledMarkets = []string{"huobi", "binance", "ftx", "mxc", "gate", "okex", "kucoin", "hotbit", "bittrex", "bithumb global", "cointiger", "bkex", "bitforex", "bitmax", "gemini", "bitfinex", "poloniex", "coinxpro", "bilaxy"}

func run() {
	collect()
	result := reconciliation()

	for val, i := range result {
		highest, hMarket := findHigh(i)
		lowest, lMarket := findLow(i)
		if highest/lowest > ratio {
			message := fmt.Sprint("ARBITRAGE FOUND!%0A<b><i>", val+"</i></b>%0A", "highest:<b> ", hMarket+": </b>", highest, "%0Alowest:<b> ", lMarket+"</b>: ", lowest)
			fmt.Println(message)
			get("https://api.telegram.org/bot" + telegramBotApi + "/sendMessage?chat_id=" + telegramChannel + "&text=" + message + "&parse_mode=html")
		}
	}
}

func main() {
	for {
		run()
		excList = nil
		exc = nil
		fmt.Println("Sleeping..")
		fmt.Println("Waking up at", time.Now().Add(5*time.Minute))
		time.Sleep(5 * time.Minute)
	}
}

func findHigh(data []map[string]float64) (float64, string) {
	var high float64
	var highMarket string
	for _, i := range data {
		for val, j := range i {
			if high == 0 {
				high = j
				highMarket = val
				continue
			}
			if high > j {
				continue
			} else {
				high = j
				highMarket = val
			}
		}
	}
	return high, highMarket
}
func findLow(data []map[string]float64) (float64, string) {
	var low float64
	var lowMarket string
	for _, i := range data {
		for val, j := range i {
			if low == 0 {
				low = j
				lowMarket = val
				continue
			}
			if low < j {
				continue
			} else {
				low = j
				lowMarket = val
			}
		}
	}
	return low, lowMarket
}
func isEnabled(data string) bool {
	data = strings.ToLower(data)
	for _, mrkts := range enabledMarkets {
		if data == mrkts {
			return true
		}
	}
	return false
}

func reconciliation() map[string][]map[string]float64 {
	// struct daki verileri coin/usdt seklinde map e yaziyorum
	// map içinde market = price seklinde map tutuyor
	coins := map[string][]map[string]float64{}
	for _, excs := range exc {
		if excs.Name == "" {
			continue
		}
		for _, i := range excs.Tickers {
			if i.Target != pair {
				continue
			}
			if i.Volume < minVolume {
				continue
			}
			if !isEnabled(excs.Name) {
				continue
			}
			tempExc := map[string]float64{}
			tempExc[excs.Name] = i.Last
			coins[i.Base+"/"+i.Target] = append(coins[i.Base+"/"+i.Target], tempExc)
		}
	}
	return coins
}

func collect() {
	log.Printf("collection started")
	start := time.Now()
	result := get("https://api.coingecko.com/api/v3/exchanges/list")
	_ = json.Unmarshal(result, &excList)

	rate := 9

	for i := 0; i < len(excList); i++ {
		var excTemp Exc
		prices := get("https://api.coingecko.com/api/v3/exchanges/" + excList[i].ID)
		_ = json.Unmarshal(prices, &excTemp)
		exc = append(exc, excTemp)
		if j := i % rate; j == 0 {
			time.Sleep(1 * time.Second)
		}
	}
	elapsed := time.Since(start)
	log.Printf("collection time: %s", elapsed)
}

func get(url string) []byte {
	resp, err := http.Get(url)
	if err != nil {
		log.Fatalf("Error fetching url %s: %v", url, err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Error fetching url %s: %v", url, err)
	}
	return body
}
