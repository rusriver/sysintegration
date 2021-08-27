package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"strings"
	"time"

	"github.com/goburrow/modbus"
	"github.com/nsqio/go-nsq"
	"github.com/olebedev/config"
)

var (
	allData    = make(map[string]float64)
	config_nsq = nsq.NewConfig()
)

func set(key string, value float64) {
	allData[key] = value
}

func periodicFunc(tick time.Time, client modbus.Client, nsq_prod string, pl []string, start_bit int, stop_bit int) {
	nums := [100]float64{}
	for i := start_bit; i < stop_bit; i++ {
		result, err := client.ReadInputRegisters(uint16(i), 4)
		value := binary.BigEndian.Uint16(result)
		nums[i] = float64(value) / 10
		if err != nil {
			log.Fatalf("%v", err)
		}
	}

	data := nums[start_bit:stop_bit]
	for i := range pl {
		set(pl[i], data[i])
	}
	p, err := nsq.NewProducer(nsq_prod, config_nsq)
	if err != nil {
		log.Panic(err)
	}
	payload, err := json.Marshal(allData)
	if err != nil {
		log.Println(err)
	}
	err = p.Publish("My_NSQ_Topic", payload)
	if err != nil {
		log.Panic(err)
	}
}

func main() {
	file, err := ioutil.ReadFile("config.yml")
	if err != nil {
		panic(err)
	}
	yamlString := string(file)

	cfg, err := config.ParseYaml(yamlString)
	ip, err := cfg.String("production.weintek.ip")
	var client = modbus.TCPClient(ip)
	nsq_prod, err := cfg.String("production.nsq.producer")
	plant_data, err := cfg.String("production.weintek.variables")
	pl := strings.Split(plant_data, ",")
	start_bit, err := cfg.Int("production.weintek.start_bit")
	stop_bit, err := cfg.Int("production.weintek.stop_bit")
	fmt.Println(start_bit, stop_bit)

	for t := range time.NewTicker(1 * time.Second).C {
		periodicFunc(t, client, nsq_prod, pl, start_bit, stop_bit)
	}
}
