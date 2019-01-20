package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"../config"
)

const programName = "bfx"

type rpcCommand struct {
	Command   string `json:"command"`
	Parameter string `json:"parameter"`
}

type devs struct {
	STATUS []struct {
		STATUS      string `json:"STATUS"`
		When        int    `json:"When"`
		Code        int    `json:"Code"`
		Msg         string `json:"Msg"`
		Description string `json:"Description"`
	} `json:"STATUS"`
	DEVS []struct {
		PGA                 int     `json:"PGA"`
		Name                string  `json:"Name"`
		ID                  int     `json:"ID"`
		Enabled             string  `json:"Enabled"`
		Status              string  `json:"Status"`
		DeviceElapsed       int     `json:"Device Elapsed"`
		MHSAv               float64 `json:"MHS av"`
		MHS20S              float64 `json:"MHS 20s"`
		MHSRolling          float64 `json:"MHS rolling"`
		Accepted            int     `json:"Accepted"`
		Rejected            int     `json:"Rejected"`
		HardwareErrors      int     `json:"Hardware Errors"`
		Utility             float64 `json:"Utility"`
		Stale               int     `json:"Stale"`
		LastSharePool       int     `json:"Last Share Pool"`
		LastShareTime       int     `json:"Last Share Time"`
		TotalMH             float64 `json:"Total MH"`
		Diff1Work           float64 `json:"Diff1 Work"`
		WorkUtility         float64 `json:"Work Utility"`
		DifficultyAccepted  float64 `json:"Difficulty Accepted"`
		DifficultyRejected  float64 `json:"Difficulty Rejected"`
		DifficultyStale     float64 `json:"Difficulty Stale"`
		LastValidWork       int     `json:"Last Valid Work"`
		DeviceHardware      float64 `json:"Device Hardware%"`
		DeviceRejected      float64 `json:"Device Rejected%"`
		FanSpeed            float64 `json:"Fan Speed"`
		FanPercent          float64 `json:"Fan Percent"`
		GPUClock            float64 `json:"GPU Clock"`
		MemoryClock         float64 `json:"Memory Clock"`
		GPUVoltage          float64 `json:"GPU Voltage"`
		GPUActivity         float64 `json:"GPU Activity"`
		Powertune           float64 `json:"Powertune"`
		Intensity           string  `json:"Intensity"`
		OCLThreads          int     `json:"OCLThreads"`
		CIntensity          float64 `json:"CIntensity"`
		XIntensity          float64 `json:"XIntensity"`
		LastShareDifficulty float64 `json:"Last Share Difficulty,omitempty"`
	} `json:"DEVS"`
	ID int `json:"id"`
}

var (
	minerGpuHashRate = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "miner_gpu_hashrate",
			Help: "Current hash rate of a GPU.",
		},
		[]string{"miner", "gpu", "symbol"},
	)

	cfg *config.Config
)

func init() {
	cfg = config.NewConfig(programName)

	// Metrics have to be registered to be exposed:
	//prometheus.MustRegister(minerTotalHashRate)
	prometheus.MustRegister(minerGpuHashRate)

}

func main() {

	go func() {
		for {
			conn, err := net.Dial("tcp", cfg.Miner.Address())

			if err == nil {

				var bytes []byte
				bytes, err = json.Marshal(rpcCommand{Command: "devs"})

				if err == nil {
					fmt.Fprintf(conn, string(bytes))

					var resp devs
					err = json.NewDecoder(bufio.NewReader(conn)).Decode(&resp)

					if err != nil {
						log.Printf("Error decoding response: %v\n", err)
						// But we're still trying to do it as well as we can.
					}
					log.Printf("Response:\n%v\n", resp)

					for i, device := range resp.DEVS {
						log.Printf("%v Device %v Hashrate %v\n", i, device.ID, device.MHS20S)

						minerGpuHashRate.With(prometheus.Labels{
							"miner":  cfg.Miner.Program(),
							"gpu":    fmt.Sprintf("GPU%d", device.ID),
							"symbol": cfg.Miner.Symbol(),
						}).Set(device.MHS20S)
					}
				} else {
					log.Printf("Error marshaling command: %v\n", err)
				}
			} else {
				log.Printf("Error connecting to miner: %v\n", err)
			}

			time.Sleep(time.Second * cfg.QueryDelay())
		}
	}()

	// The Handler function provides a default handler to expose metrics
	// via an HTTP server. "/metrics" is the usual endpoint for that.
	http.Handle("/metrics", promhttp.Handler())
	log.Fatal(http.ListenAndServe(cfg.Prometheus.Address(), nil))
}
