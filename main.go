package main

import (
	"crypto/hmac"
	"crypto/sha256"
	b64 "encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var FTX_API_SECRET = ""
var FTX_API_KEY = ""
var KUCOIN_API_SECRET = ""
var KUCOIN_API_KEY = ""
var KUCOIN_API_PASSPHRASE = ""
var BINANCE_API_KEY = ""

func main() {
	http.HandleFunc("/", getPrices)
	http.ListenAndServe(":8090", nil)
}

func getPrices(w http.ResponseWriter, req *http.Request) {
	result1 := make(chan map[string]string)
	go getFtxData(result1)
	ftxData := <-result1

	result2 := make(chan map[string]string)
	go getMexcData(result2)
	mexcData := <-result2

	result3 := make(chan map[string]string)
	go getKucoinData(result3)
	kucoinData := <-result3

	result4 := make(chan map[string]string)
	go getGateioData(result4)
	gateioData := <-result4

	result5 := make(chan map[string]string)
	go getBinanceData(result5)
	binanceData := <-result5

	jsonStr, err := json.Marshal(merge(ftxData, mexcData, kucoinData, gateioData, binanceData))

	if err != nil {
		http.NotFound(w, req)
	} else {
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonStr)
	}
}

func merge(ms ...map[string]string) map[string]string {
	res := map[string]string{}
	for _, m := range ms {
		for k, v := range m {
			res[k] = v
		}
	}
	return res
}

func filterData(symbol string) bool {
	return true
}

func calculateHmac(payload string, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))

	return hex.EncodeToString(mac.Sum(nil))
}

func getFtxData(result chan map[string]string) {
	var tickers = make(map[string]string)

	url := "https://ftx.com/api/markets"
	method := "GET"
	ts := strconv.FormatInt(time.Now().UTC().Unix()*1000, 10)
	signature := calculateHmac(ts+method+url, FTX_API_SECRET)

	var headers = map[string]string{
		"Content-Type": "application/json",
		"FTX-KEY":      FTX_API_KEY,
		"FTX-SIGN":     signature,
		"FTX-TS":       ts,
	}

	responseData := sendRequest("GET", url, headers)

	var responseObject struct {
		Result []struct {
			Symbol string  `json:"name"`
			Price  float64 `json:"last"`
		} `json:"result"`
	}

	json.Unmarshal(responseData, &responseObject)

	for _, p := range responseObject.Result {
		if filterData(p.Symbol) {
			arrayKey := strings.Replace(p.Symbol, "/", "", 1)

			if strings.HasSuffix(arrayKey, "USD") {
				arrayKey = arrayKey + "T"
			}

			tickers[arrayKey] = fmt.Sprint(p.Price)
		}
	}

	result <- tickers
}

func getKucoinData(result chan map[string]string) {
	var tickers = make(map[string]string)

	url := "https://api.kucoin.com"
	path := "/api/v1/market/allTickers"
	method := "GET"
	ts := strconv.FormatInt(time.Now().UTC().Unix()*1000, 10)
	signature := b64.StdEncoding.EncodeToString([]byte(calculateHmac(ts+method+path, KUCOIN_API_SECRET)))

	var headers = map[string]string{
		"KC-API-SIGN":       signature,
		"KC-API-TIMESTAMP":  ts,
		"KC-API-KEY":        KUCOIN_API_KEY,
		"KC-API-PASSPHRASE": KUCOIN_API_PASSPHRASE,
	}

	responseData := sendRequest("GET", url+path, headers)

	var responseObject struct {
		RawData struct {
			Coins []struct {
				Symbol string `json:"symbol"`
				Price  string `json:"last"`
			} `json:"ticker"`
		} `json:"data"`
	}

	json.Unmarshal(responseData, &responseObject)

	for _, p := range responseObject.RawData.Coins {
		if filterData(p.Symbol) {
			arrayKey := strings.Replace(p.Symbol, "-", "", 1)

			if strings.HasSuffix(arrayKey, "USD") {
				arrayKey = arrayKey + "T"
			}

			tickers[arrayKey] = p.Price
		}
	}

	result <- tickers
}

func getMexcData(result chan map[string]string) {
	var tickers = make(map[string]string)

	responseData := sendRequest("GET", "https://www.mexc.com/open/api/v2/market/ticker", make(map[string]string))

	var responseObject struct {
		Coins []struct {
			Symbol string `json:"symbol"`
			Price  string `json:"last"`
		} `json:"data"`
	}

	json.Unmarshal(responseData, &responseObject)

	for _, p := range responseObject.Coins {
		if filterData(p.Symbol) {
			arrayKey := strings.Replace(p.Symbol, "_", "", 1)

			if strings.HasSuffix(arrayKey, "USD") {
				arrayKey = arrayKey + "T"
			}

			tickers[arrayKey] = p.Price
		}
	}

	result <- tickers
}

func getGateioData(result chan map[string]string) {
	var tickers = make(map[string]string)

	var headers = map[string]string{
		"Content-Type": "application/json",
		"Accept":       "application/json",
	}

	responseData := sendRequest("GET", "https://api.gateio.ws/api/v4/spot/tickers", headers)

	var responseObject []struct {
		Symbol string `json:"currency_pair"`
		Price  string `json:"last"`
	}

	json.Unmarshal(responseData, &responseObject)

	for _, p := range responseObject {
		if filterData(p.Symbol) {
			arrayKey := strings.Replace(p.Symbol, "_", "", 1)

			if strings.HasSuffix(arrayKey, "USD") {
				arrayKey = arrayKey + "T"
			}

			tickers[arrayKey] = p.Price
		}
	}

	result <- tickers
}

func getBinanceData(result chan map[string]string) {
	var tickers = make(map[string]string)

	var headers = map[string]string{
		"Content-Type": "application/json",
		"X-MBX-APIKEY": BINANCE_API_KEY,
	}

	responseData := sendRequest("GET", "https://api.binance.com/api/v3/ticker/price", headers)

	var responseObject []struct {
		Symbol string `json:"symbol"`
		Price  string `json:"price"`
	}

	json.Unmarshal(responseData, &responseObject)

	for _, p := range responseObject {
		if filterData(p.Symbol) {
			arrayKey := strings.Replace(p.Symbol, "_", "", 1)

			if strings.HasSuffix(arrayKey, "USD") {
				arrayKey = arrayKey + "T"
			}

			tickers[arrayKey] = p.Price
		}
	}

	result <- tickers
}

func sendRequest(method string, url string, headers map[string]string) []byte {
	client := http.Client{}

	req, _ := http.NewRequest(method, url, nil)

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := client.Do(req)
	if err != nil {
		return make([]byte, 0)
	}

	responseData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return make([]byte, 0)
	}

	return responseData
}
