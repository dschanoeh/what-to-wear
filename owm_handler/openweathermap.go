package owm_handler

import (
	"errors"

	owm "github.com/dschanoeh/go-owm"
	log "github.com/sirupsen/logrus"
)

type OpenWeatherMapConfig struct {
	APIKey    string  `yaml:"api_key"`
	Latitude  float64 `yaml:"latitude"`
	Longitude float64 `yaml:"longitude"`
	Language  string  `yaml:"language"`
}

type WeatherReport struct {
	WeatherIconURL  string
	Description     string
	FontAwesomeIcon string
}

func GetData(config OpenWeatherMapConfig) (*owm.WeatherData, *WeatherReport, error) {
	weather, err := owm.GetWeather(config.Latitude, config.Longitude, config.APIKey)

	if err != nil {
		return nil, nil, err
	}

	if len(weather.Current.Weather) < 1 {
		return nil, nil, errors.New("No current weather received")
	}
	currentWeather := weather.Current.Weather[0]

	report := WeatherReport{}
	report.Description = currentWeather.Description
	report.WeatherIconURL = "http://openweathermap.org/img/wn/" + currentWeather.Icon + "@2x.png"
	report.FontAwesomeIcon = FontAwesomeIconFromWeatherID(currentWeather.ID)

	return weather, &report, nil
}

// FontAwesomeIconFromWeatherID returns a font awesome icon matching a owm weather condition
// See https://openweathermap.org/weather-conditions for an overview
func FontAwesomeIconFromWeatherID(id uint16) string {
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
	case 701, 711, 721, 731, 741, 751, 761, 762, 771, 781:
		return "smog"
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
