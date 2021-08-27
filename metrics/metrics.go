package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"sync"

	"github.com/nsqio/go-nsq"

	"github.com/olebedev/config"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	p2 = make(map[string]float64)
)

func gauge(id []prometheus.Gauge, prometheus_var []string) {
	for i := range prometheus_var {
		id[i] = prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: prometheus_var[i],
			})
		prometheus.MustRegister(id[i])
	}
}

func main() {

	file, err := ioutil.ReadFile("config.yml")
	if err != nil {
		panic(err)
	}
	yamlString := string(file)

	cfg, err := config.ParseYaml(yamlString)
	nsq_address, err := cfg.String("production.nsq.nsq_address")
	data, err := cfg.String("production.prometheus.variables")
	prometheus_var := strings.Split(data, ",")
	port, err := cfg.String("production.prometheus.port")
	tab, err := cfg.String("production.prometheus.tab")

	flag.Parse()

	var temp_separ, temp_heater, temp_out, temp_in, temp_led, tank_level, flow prometheus.Gauge
	id := []prometheus.Gauge{temp_separ, temp_heater, temp_out, temp_in, temp_led, tank_level, flow}
	gauge(id, prometheus_var)

	go func() {
		for {
			wg := &sync.WaitGroup{}
			wg.Add(1)

			decodeConfig := nsq.NewConfig()
			c, err := nsq.NewConsumer("My_NSQ_Topic", "My_NSQ_Channel", decodeConfig)
			if err != nil {
				log.Panic("Could not create consumer")
			}

			c.AddHandler(nsq.HandlerFunc(func(message *nsq.Message) error {
				log.Println("NSQ message received:")
				log.Println(string(message.Body))
				err1 := json.Unmarshal(message.Body, &p2)
				log.Print(err1)
				for i := range prometheus_var {
					id[i].Set(float64(p2[prometheus_var[i]]))
				}
				return nil
			}))

			err = c.ConnectToNSQD(nsq_address)
			if err != nil {
				log.Panic("Could not connect")
			}
			log.Println("Awaiting messages from NSQ topic \"My NSQ Topic\"...")
			wg.Wait()
		}
	}()

	http.Handle(tab, promhttp.Handler())
	http.ListenAndServe(port, nil)
}
