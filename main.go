package main

import (
	"flag"
	"fmt"
	"html/template"
	"io/ioutil"
	"os"
	"time"

	"github.com/dschanoeh/what-to-wear/evaluator"
	"github.com/dschanoeh/what-to-wear/imaging"
	"github.com/dschanoeh/what-to-wear/mqtt"
	"github.com/dschanoeh/what-to-wear/owm_handler"
	"github.com/dschanoeh/what-to-wear/server"
	"github.com/robfig/cron/v3"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
	builtBy = "unknown"

	config         = Config{}
	cronScheduler  = cron.New()
	webServer      *server.Server
	imageProcessor *imaging.ImageProcessor
	mqttClient     *mqtt.MQTTClient
)

type Config struct {
	OpenWeatherMap owm_handler.OpenWeatherMapConfig `yaml:"open_weather_map"`
	Messages       []evaluator.Message              `yaml:"messages"`
	ServerConfig   server.ServerConfig              `yaml:"server"`
	CronExpression string                           `yaml:"cron_expression"`
	ImageConfig    imaging.ImageConfig              `yaml:"imaging"`
	MQTTConfig     mqtt.MQTTConfig                  `yaml:"mqtt"`
}

func main() {
	var verbose = flag.Bool("verbose", false, "Turns on verbose information on the update process. Otherwise, only errors cause output.")
	var debug = flag.Bool("debug", false, "Turns on debug information")
	var configFile = flag.String("config", "", "Config file")
	var versionFlag = flag.Bool("version", false, "Prints version information of the hover-ddns binary")

	flag.Parse()

	if *versionFlag {
		fmt.Printf("hover-ddns version %s, commit %s, built at %s by %s\n", version, commit, date, builtBy)
		os.Exit(0)
	}

	if *verbose {
		log.SetLevel(log.InfoLevel)
	} else if *debug {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.ErrorLevel)
	}

	if *configFile == "" {
		log.Error("Please provide a config file to read")
		flag.Usage()
		os.Exit(1)
	}

	err := loadConfig(*configFile, &config)
	if err != nil {
		log.Error("Could not load config file: ", err)
		os.Exit(1)
	}

	err = evaluator.Compile(&config.Messages)
	if err != nil {
		log.Error("Could not compile messages: ", err)
		os.Exit(1)
	}
	webServer = server.New(config.ServerConfig)
	imageProcessor = imaging.New(&config.ImageConfig)
	mqttClient, err = mqtt.New(&config.MQTTConfig)
	if err != nil {
		log.Error("Error creating MQTT client: ", err)
		os.Exit(1)
	}

	// Schedule future periodic update calls
	_, err = cronScheduler.AddFunc(config.CronExpression, updateData)
	if err != nil {
		log.Error("Was not able to schedule periodic execution: ", err)
		os.Exit(1)
	}
	cronScheduler.Start()

	// Update once so data is available to be served
	updateData()

	// Now let's serve
	webServer.Serve()
}

func updateData() {
	log.Info("Updating data...")
	data, report := owm_handler.GetData(config.OpenWeatherMap)
	log.Info(data)
	messages := evaluator.Evaluate(data, &config.Messages)

	// Convert to HTML templates to allow HTML tags to pass through
	templateMessages := make([]template.HTML, len(messages))
	for i := range messages {
		templateMessages[i] = template.HTML(messages[i])
	}

	currentDateString := time.Now().Format(time.RFC850)
	tillNextUpdate := 0
	if len(cronScheduler.Entries()) > 0 {
		nextTrigger := cronScheduler.Entries()[0].Next
		delta := nextTrigger.Sub(time.Now())
		tillNextUpdate = int(delta.Seconds())
	} else {
		log.Warn("Scheduler doesn't seem to have any entries...")
	}

	content := server.Content{
		Messages:       templateMessages,
		Version:        version,
		CreationTime:   currentDateString,
		City:           config.OpenWeatherMap.City,
		WeatherIconURL: report.WeatherIconURL,
		WeatherReport:  fmt.Sprintf("%.0f°C", data.CurrentTemp) + " - " + report.Description,
	}

	webServer.UpdateData(&content)
	imageProcessor.Update()
	mqttClient.Post(imageProcessor.GetImageAsBinary(), currentDateString, tillNextUpdate)
}

func loadConfig(filename string, config *Config) error {
	yamlFile, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}

	err = yaml.Unmarshal(yamlFile, &config)
	if err != nil {
		return err
	}

	return nil
}
