package main

import (
        "encoding/json"
        "log"
        "net/http"
        "os"
        "strconv"
        "time"

        "github.com/prometheus/client_golang/prometheus"
        "github.com/prometheus/client_golang/prometheus/promhttp"
)

type blockchainStats struct {
        Data struct {
                Attributes struct {
                        TipBlockNumber string `json:"tip_block_number"`
                } `json:"attributes"`
        } `json:"data"`
}

func main() {
        // 获取 CKB 浏览器地址变量
        browserURL := os.Getenv("BROWSER_URL")
        if browserURL == "" {
                log.Fatalf("BROWSER_URL environment variable not set")
        }

        // 获取抓取时间变量
        timeoutStr := os.Getenv("TIMEOUT")
        if timeoutStr == "" {
                timeoutStr = "30s" // 默认 30 秒抓取一次
        }

        timeout, err := time.ParseDuration(timeoutStr)
        if err != nil {
                log.Fatalf("Failed to parse timeout duration: %v", err)
        }

        // 注册自定义指标
        blockchainHeight := prometheus.NewGauge(prometheus.GaugeOpts{
                Name: "blockchain_height",
                Help: "The current height of the blockchain",
        })

        prometheus.MustRegister(blockchainHeight)

        // 注册接口状态指标
        apiStatus := prometheus.NewGauge(prometheus.GaugeOpts{
                Name: "api_status",
                Help: "The status of the API endpoint",
        })

        prometheus.MustRegister(apiStatus)

        // 每隔一段时间抓取区块高度数据并更新指标
        go func() {
                for {
                        client := &http.Client{
                                Timeout: 5 * time.Second, // 设置请求超时时间
                        }

                        req, err := http.NewRequest("GET", browserURL+"/api/v1/statistics", nil)
                        if err != nil {
                                log.Printf("Failed to create request: %v", err)
                                continue
                        }

                        req.Header.Set("Content-Type", "application/vnd.api+json")
                        req.Header.Set("Accept", "application/vnd.api+json")

                        resp, err := client.Do(req)
                        if err != nil {
                                log.Printf("Failed to fetch data from CKB browser: %v", err)
                                apiStatus.Set(0) // 设置接口状态为失败
                                continue
                        }
                        defer resp.Body.Close()

                        var stats blockchainStats
                        err = json.NewDecoder(resp.Body).Decode(&stats)
                        if err != nil {
                                log.Printf("Failed to parse response from CKB browser: %v", err)
                                apiStatus.Set(0) // 设置接口状态为失败
                                continue
                        }

                        blockNumberStr := stats.Data.Attributes.TipBlockNumber
                        blockNumber, err := strconv.ParseFloat(blockNumberStr, 64)
                        if err != nil {
                                log.Printf("Failed to convert block number to float64: %v", err)
                                apiStatus.Set(0) // 设置接口状态为失败
                                continue
                        }

                        log.Printf("Fetched blockchain height: %s, response status: %s", blockNumberStr, resp.Status)
                        blockchainHeight.Set(blockNumber)
                        apiStatus.Set(1) // 设置接口状态为成功

                        time.Sleep(timeout)
                }
        }()

        // 启动 HTTP 服务器并暴露指
        // 启动 HTTP 服务器并暴露指标
        http.Handle("/metrics", promhttp.Handler())
        log.Fatal(http.ListenAndServe(":8080", nil))
}
