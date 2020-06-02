package owm_handler

import (
	owm "github.com/briandowns/openweathermap"
	log "github.com/sirupsen/logrus"
)

type OpenWeatherMapConfig struct {
	APIKey   string `yaml:"api_key"`
	City     string `yaml:"city"`
	Language string `yaml:"language"`
}

type EvaluationData struct {
	CurrentTemp float64
	TempMin     float64
	TempMax     float64
	FeelsLike   float64
	Rain1h      float64
	Rain3h      float64
	Snow1h      float64
	Snow3h      float64
	UVValue     float64
	Cloudiness  int
	WindSpeed   float64
}

type WeatherReport struct {
	WeatherIconURL string
	Description    string
}

func GetData(config OpenWeatherMapConfig) (*EvaluationData, *WeatherReport, error) {
	w, err := owm.NewCurrent("C", config.Language, config.APIKey)

	if err != nil {
		return nil, nil, err
	}

	w.CurrentByName(config.City)
	log.Debugf("OWM Weather: %+v\n", w)

	weather := w.Weather[0]

	// Get UV info
	uv, err := owm.NewUV(config.APIKey)
	err = uv.Current(&w.GeoPos)
	if err != nil {
		return nil, nil, err
	}
	log.Debugf("OWM UV: %+v\n", uv)
	uvI, err := uv.UVInformation()
	uvInfo := uvI[0]
	log.Debugf("OWM UV Info: %+v\n", uvInfo)

	data := EvaluationData{}
	data.CurrentTemp = w.Main.Temp
	data.TempMin = w.Main.TempMin
	data.TempMax = w.Main.TempMax
	data.Cloudiness = w.Clouds.All
	data.WindSpeed = w.Wind.Speed
	data.FeelsLike = w.Main.FeelsLike
	data.Rain1h = w.Rain.OneH
	data.Rain3h = w.Rain.ThreeH
	data.Snow1h = w.Snow.OneH
	data.Snow3h = w.Snow.ThreeH
	data.UVValue = uv.Value

	report := WeatherReport{}
	report.Description = weather.Description
	report.WeatherIconURL = "http://openweathermap.org/img/wn/" + weather.Icon + "@2x.png"

	return &data, &report, nil
}
