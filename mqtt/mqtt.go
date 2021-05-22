package mqtt

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	log "github.com/sirupsen/logrus"
)

const (
	ConnectTimeout = time.Second * 60
)

type MQTTConfig struct {
	BrokerURL string `yaml:"broker_url"`
	BaseTopic string `yaml:"base_topic"`
	ChunkSize int    `yaml:"chunk_size"`
}

type MQTTClient struct {
	config  *MQTTConfig
	client  mqtt.Client
	options *mqtt.ClientOptions
}

func New(config *MQTTConfig) (*MQTTClient, error) {
	c := MQTTClient{config: config}

	c.options = mqtt.NewClientOptions()
	c.options.AddBroker(config.BrokerURL)
	c.options.SetAutoReconnect(true)
	c.options.SetConnectionLostHandler(connectionLostHandler)
	c.options.SetReconnectingHandler(reconnectingHandler)
	c.options.SetConnectTimeout(ConnectTimeout)
	c.options.SetConnectRetry(true)
	c.options.SetConnectRetryInterval(time.Second * 5)
	c.client = mqtt.NewClient(c.options)
	token := c.client.Connect()

	if !token.WaitTimeout(ConnectTimeout) {
		return nil, errors.New("could not connect MQTT client")
	}
	log.Info("MQTT connected")

	return &c, nil
}

func reconnectingHandler(client mqtt.Client, options *mqtt.ClientOptions) {
	log.Info("Attempting to reconnect to broker...")
}

func connectionLostHandler(client mqtt.Client, reason error) {
	log.Warn("MQTT broker connection lost: ", reason.Error())
}

func (c *MQTTClient) Close() error {
	c.client.Disconnect(100)
	return nil
}

func (c *MQTTClient) Post(payload []byte, currentDateString string) error {
	if !c.client.IsConnected() {
		return errors.New("MQTT not connected")
	}

	c.client.Publish(fmt.Sprintf("%s/%s", c.config.BaseTopic, "generationTime"), 0, true, []byte(currentDateString))
	c.client.Publish(fmt.Sprintf("%s/%s", c.config.BaseTopic, "data"), 0, true, payload)

	if c.config.ChunkSize != 0 {
		if len(payload)%c.config.ChunkSize != 0 {
			log.Error("Data length is no multiple of the chunk size. This is likely a configuration error. Not posting chunks.")
		} else {
			num := len(payload) / c.config.ChunkSize
			c.client.Publish(fmt.Sprintf("%s/%s", c.config.BaseTopic, "numChunks"), 0, true, []byte(strconv.Itoa(num)))
			for i := 0; i < num; i++ {
				chunk := payload[c.config.ChunkSize*i : c.config.ChunkSize*i+c.config.ChunkSize-1]
				c.client.Publish(fmt.Sprintf("%s/%s/%d", c.config.BaseTopic, "chunks", i), 0, true, chunk)
			}
		}
	}

	return nil
}

func (c *MQTTClient) PostImageURL(url string) error {
	if !c.client.IsConnected() {
		return errors.New("MQTT not connected")
	}

	c.client.Publish(fmt.Sprintf("%s/%s", c.config.BaseTopic, "rawImageURL"), 0, true, []byte(url))

	return nil
}

func (c *MQTTClient) RefreshUpdateTime(tillNextUpdate int) error {
	if !c.client.IsConnected() {
		return errors.New("MQTT not connected")
	}

	c.client.Publish(fmt.Sprintf("%s/%s", c.config.BaseTopic, "nextUpdateIn"), 0, true, []byte(strconv.Itoa(tillNextUpdate)))

	return nil
}
