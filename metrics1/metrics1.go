package main

import (
	"encoding/json"
	"flag"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/nsqio/go-nsq"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

//var addr = flag.String("listen-address", ":8080",
//	"The address to listen on for HTTP requests.")

var (
	p2 = make(map[string]float64)
)

func main() {
	flag.Parse()

	usersRegistered := prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "users_registered",
		})
	prometheus.MustRegister(usersRegistered)

	usersOnline := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "users_online",
		})
	prometheus.MustRegister(usersOnline)

	temp_separ := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "temp_separ",
		})
	prometheus.MustRegister(temp_separ)

	temp_heater := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "temp_heater",
		})
	prometheus.MustRegister(temp_heater)

	temp_out := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "temp_out",
		})
	prometheus.MustRegister(temp_out)

	temp_in := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "temp_in",
		})
	prometheus.MustRegister(temp_in)

	temp_led := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "temp_led",
		})
	prometheus.MustRegister(temp_led)

	tank_level := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "tank_level",
		})
	prometheus.MustRegister(tank_level)

	flow := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "flow",
		})
	prometheus.MustRegister(flow)

	requestProcessingTimeSummaryMs := prometheus.NewSummary(
		prometheus.SummaryOpts{
			Name:       "request_processing_time_summary_ms",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		})
	prometheus.MustRegister(requestProcessingTimeSummaryMs)

	requestProcessingTimeHistogramMs := prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "request_processing_time_histogram_ms",
			Buckets: prometheus.LinearBuckets(0, 10, 20),
		})
	prometheus.MustRegister(requestProcessingTimeHistogramMs)

	go func() {
		for {
			usersRegistered.Inc() // or: Add(5)
			time.Sleep(1000 * time.Millisecond)
		}
	}()

	go func() {
		for {
			for i := 0; i < 10000; i++ {
				usersOnline.Set(float64(i)) // or: Inc(), Dec(), Add(5), Dec(5)
				time.Sleep(10 * time.Millisecond)
			}
		}
	}()

	go func() {
		for {
			wg := &sync.WaitGroup{}
			wg.Add(1)

			decodeConfig := nsq.NewConfig()
			c, err := nsq.NewConsumer("My_NSQ_Topic", "My_NSQ_Channel", decodeConfig)
			if err != nil {
				log.Panic("Could not create consumer")
			}
			//c.MaxInFlight defaults to 1

			c.AddHandler(nsq.HandlerFunc(func(message *nsq.Message) error {
				log.Println("NSQ message received:")
				log.Println(string(message.Body))
				err1 := json.Unmarshal(message.Body, &p2)
				log.Print(err1)
				temp_separ.Set(float64(p2["T_separ"]))
				temp_heater.Set(float64(p2["T_heater"]))
				temp_in.Set(float64(p2["T_in"]))
				temp_out.Set(float64(p2["T_out"]))
				temp_led.Set(float64(p2["T_led"]))
				flow.Set(float64(p2["Flow"]))
				tank_level.Set(float64(p2["Level"]))
				return nil
			}))

			err = c.ConnectToNSQD("127.0.0.1:4150")
			if err != nil {
				log.Panic("Could not connect")
			}
			log.Println("Awaiting messages from NSQ topic \"My NSQ Topic\"...")
			wg.Wait()
		}
	}()

	go func() {
		src := rand.NewSource(time.Now().UnixNano())
		rnd := rand.New(src)
		for {
			obs := float64(100 + rnd.Intn(30))
			requestProcessingTimeSummaryMs.Observe(obs)
			requestProcessingTimeHistogramMs.Observe(obs)
			time.Sleep(10 * time.Millisecond)
		}
	}()

	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(":2112", nil)
}
