package main

import (
	"flag"
	"fmt"
	"html/template"
	"io/ioutil"
	"os"
	"os/signal"
	"syscall"
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

const (
	notificationTimeDelay = 5 + (imaging.VirtualTimeBudget / 1000)
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
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)

	go func() {
		for {
			s := <-sigChan
			switch s {
			case syscall.SIGHUP:
				log.Info("SIGHUP")
				cleanup()
				os.Exit(0)

			case syscall.SIGINT:
				log.Info("SIGINT")
				cleanup()
				os.Exit(0)

			case syscall.SIGTERM:
				log.Info("SIGTERM")
				cleanup()
				os.Exit(0)

			case syscall.SIGQUIT:
				log.Info("SIGQUIT")
				cleanup()
				os.Exit(0)

			default:
				log.Warn("Received unknown signal")
			}
		}
	}()

	var verbose = flag.Bool("verbose", false, "Turns on verbose information on the update process. Otherwise, only errors cause output.")
	var debug = flag.Bool("debug", false, "Turns on debug information")
	var configFile = flag.String("config", "", "Config file")
	var versionFlag = flag.Bool("version", false, "Prints version information of this binary")

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
	imageProcessor, err = imaging.New(&config.ImageConfig)
	if err != nil {
		log.Error("Error creating image processor: ", err)
		os.Exit(1)
	}
	defer imageProcessor.Close()
	mqttClient, err = mqtt.New(&config.MQTTConfig)
	if err != nil {
		log.Error("Error creating MQTT client: ", err)
		os.Exit(1)
	}
	mqttClient.PostImageURL("http://" + config.ServerConfig.Listen + "/eInkImage")

	// Schedule future periodic update calls
	_, err = cronScheduler.AddFunc(config.CronExpression, updateData)
	if err != nil {
		log.Error("Was not able to schedule periodic execution: ", err)
		os.Exit(1)
	}
	cronScheduler.Start()

	// Update once so data is available to be served
	go updateData()

	// Start computing and publishing update times
	go publishNextUpdateTime()

	// Now let's serve
	webServer.Serve()
}

func cleanup() {
	log.Info("Cleaning up...")
	cronScheduler.Stop()
	imageProcessor.Close()
	mqttClient.Close()
	webServer.Close()
}

func updateData() {
	log.Info("Updating data...")
	data, report, err := owm_handler.GetData(config.OpenWeatherMap)
	if err != nil {
		log.Error("Didn't receive updated information. Skipping update: ", err)
		return
	}
	log.Infof("Evaluation data: %+v\n", data)
	log.Infof("Weather report: %+v\n", report)

	messages := evaluator.Evaluate(data, &config.Messages)

	// Convert to HTML templates to allow HTML tags to pass through
	templateMessages := make([]template.HTML, len(messages))
	for i := range messages {
		templateMessages[i] = template.HTML(messages[i])
	}

	currentDateString := time.Now().Format(time.RFC850)
	content := server.Content{
		Messages:        templateMessages,
		Version:         version,
		CreationTime:    currentDateString,
		Location:        fmt.Sprintf("(%.3f, %.3f)", config.OpenWeatherMap.Latitude, config.OpenWeatherMap.Longitude),
		WeatherIconURL:  report.WeatherIconURL,
		FontAwesomeIcon: report.FontAwesomeIcon,
		WeatherReport:   fmt.Sprintf("%.0f°C", data.Current.Temperature) + " - " + report.Description,
	}

	webServer.UpdateData(&content)
	imageProcessor.Update()
	mqttClient.Post(imageProcessor.GetImageAsBinary(), currentDateString)
	webServer.UpdateImage(imageProcessor.GetImageAsBinary())
	mqttClient.PostImageURL("http://" + config.ServerConfig.Listen + "/eInkImage")
}

func publishNextUpdateTime() {
	for {
		tillNextUpdate := 0
		if len(cronScheduler.Entries()) > 0 {
			nextTrigger := cronScheduler.Entries()[0].Next
			delta := time.Until(nextTrigger)
			tillNextUpdate = int(delta.Seconds())
			// We'll lie a little bit to make sure the image is already rendered when the client checks in
			tillNextUpdate += notificationTimeDelay
		} else {
			log.Warn("Scheduler doesn't seem to have any entries...")
		}
		mqttClient.RefreshUpdateTime(tillNextUpdate)

		time.Sleep(5 * time.Second)
	}
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
