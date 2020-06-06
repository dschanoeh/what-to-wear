package owm_handler

import (
	"math"
	"time"

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
	Forecast    ForecastEvaluation
	CurrentTime time.Time
}

type ForecastEvaluation struct {
	ForecastData owm.Forecast5WeatherData
}

type WeatherReport struct {
	WeatherIconURL  string
	Description     string
	FontAwesomeIcon string
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

	// Get 5 hour forecast
	fc, err := owm.NewForecast("5", "C", config.Language, config.APIKey)
	if err != nil {
		return nil, nil, err
	}
	fc.DailyByName(config.City, 5)
	forecast := fc.ForecastWeatherJson.(*owm.Forecast5WeatherData)
	log.Debugf("OWM FC: %+v\n", forecast)

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
	data.Forecast = ForecastEvaluation{ForecastData: *forecast}

	report := WeatherReport{}
	report.Description = weather.Description
	report.WeatherIconURL = "http://openweathermap.org/img/wn/" + weather.Icon + "@2x.png"
	report.FontAwesomeIcon = FontAwesomeIconFromWeatherID(weather.ID)

	return &data, &report, nil
}

// FontAwesomeIconFromWeatherID returns a font awesome icon matching a owm weather condition
// See https://openweathermap.org/weather-conditions for an overview
func FontAwesomeIconFromWeatherID(id int) string {
	switch id {
	case 200, 201, 202, 210, 211, 212, 221, 230, 231, 232:
		return "bolt"
	case 300, 301, 302, 310, 311, 312, 313, 314, 321:
		return "cloud-rain"
	case 500, 501, 502, 503, 504:
		return "cloud-sun-rain"
	case 511:
		return "snowflake"
	case 520, 521, 522, 531:
		return "cloud-showers-heavy"
	case 600, 601, 602, 611, 612, 613, 615, 616, 620, 621, 622:
		return "snowflake"
	case 800:
		return "sun"
	case 801, 802:
		return "cloud-sun"
	case 803, 804:
		return "cloud"
	default:
		log.Warnf("Couldn't find icon for weather ID %d", id)
		return "question"
	}
}

func (fc ForecastEvaluation) TempIn(hours int) float64 {
	entry := fc.WeatherIn(hours)
	if entry == nil {
		return -1
	}
	return entry.Main.Temp
}

// WeatherIn returns the closest weather in 'hours' time from now
func (fc ForecastEvaluation) WeatherIn(hours int) *owm.Forecast5WeatherList {
	referenceTime := time.Now().Add(time.Hour * time.Duration(hours))

	for i := 0; i < len(fc.ForecastData.List); i++ {
		forecastTime := time.Unix(int64(fc.ForecastData.List[i].Dt), 0)
		difference := referenceTime.Sub(forecastTime)
		log.Debugf("Difference: %f", difference.Hours())
		if math.Abs(difference.Hours()) < 1.5 {
			log.Debugf("Forecast time %s is closest to reference time %s", forecastTime.String(), referenceTime.String())
			return &fc.ForecastData.List[i]
		}
	}

	log.Debugf("Didn't find a close forecast time")
	return nil
}

// CumulativePrecipitationTill returns the cumulative precipitation from now till hour of the day
func (fc ForecastEvaluation) CumulativePrecipitationTill(hour int) float64 {
	currentTime := time.Now()
	endtime := time.Date(currentTime.Year(), currentTime.Month(), currentTime.Day(), hour, 0, 0, 0, currentTime.Location())
	val := 0.0

	for _, item := range fc.ForecastData.List {
		forecastTime := time.Unix(int64(item.Dt), 0)
		if forecastTime.After(currentTime) && forecastTime.Before(endtime) {
			precipAmount := item.Rain.ThreeH + item.Snow.ThreeH
			log.Debugf("Time %s matches the criteria - adding %f precipitation", forecastTime.String(), precipAmount)
			val += precipAmount
		}
	}

	return val
}

// AverageTermperatureTill returns the average temperature from now till hour of the day
func (fc ForecastEvaluation) AverageTermperatureTill(hour int) float64 {
	currentTime := time.Now()
	endtime := time.Date(currentTime.Year(), currentTime.Month(), currentTime.Day(), hour, 0, 0, 0, currentTime.Location())
	val := 0.0
	num := 0

	for _, item := range fc.ForecastData.List {
		forecastTime := time.Unix(int64(item.Dt), 0)
		if forecastTime.After(currentTime) && forecastTime.Before(endtime) {
			temp := item.Main.Temp
			log.Debugf("Time %s matches the criteria - adding %f temperature", forecastTime.String(), temp)
			val += temp
			num++
		}
	}

	return val / float64(num)
}
