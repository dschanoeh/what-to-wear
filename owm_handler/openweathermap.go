package owm_handler

import (
	"fmt"
	"log"

	owm "github.com/briandowns/openweathermap"
)

type OpenWeatherMapConfig struct {
	APIKey   string `yaml:"api_key"`
	City     string `yaml:"city"`
	Language string `yaml:"language"`
}

type EvaluationData struct {
	CurrentTemp float64
	FeelsLike   float64
	Rain1h      float64
	Rain3h      float64
	Snow1h      float64
	Snow3h      float64
}

type WeatherReport struct {
	WeatherIconURL string
	Description    string
}

func GetData(config OpenWeatherMapConfig) (*EvaluationData, *WeatherReport) {
	w, err := owm.NewCurrent("C", config.Language, config.APIKey)

	if err != nil {
		log.Fatalln(err)
	}

	w.CurrentByName(config.City)
	fmt.Println(w)

	weather := w.Weather[0]

	data := EvaluationData{}
	data.CurrentTemp = w.Main.Temp
	data.FeelsLike = w.Main.FeelsLike
	data.Rain1h = w.Rain.OneH
	data.Rain3h = w.Rain.ThreeH
	data.Snow1h = w.Snow.OneH
	data.Snow3h = w.Snow.ThreeH

	report := WeatherReport{}
	report.Description = weather.Description
	report.WeatherIconURL = "http://openweathermap.org/img/wn/" + weather.Icon + "@2x.png"

	return &data, &report
}
