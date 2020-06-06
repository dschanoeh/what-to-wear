package evaluator

import (
	"errors"
	"fmt"

	"github.com/antonmedv/expr"
	"github.com/antonmedv/expr/vm"
	"github.com/dschanoeh/what-to-wear/owm_handler"
	log "github.com/sirupsen/logrus"
)

type Message struct {
	Message           string `yaml:"message"`
	Condition         string `yaml:"condition"`
	compiledCondition *vm.Program
	compiledMessage   *vm.Program
	Variables         []Variable `yaml:"variables"`
}

type Variable struct {
	Name    string   `yaml:"name"`
	Choices []Choice `yaml:"choices"`
}

type Choice struct {
	Expression string `yaml:"expression"`
	Value      string `yaml:"value"`
	program    *vm.Program
}

func buildEnv(data *owm_handler.EvaluationData) *map[string]interface{} {
	if data == nil {
		env := map[string]interface{}{
			"temperature": 20.0,
			"feelsLike":   21.0,
			"rain1h":      0.0,
			"rain3h":      0.0,
			"snow1h":      0.0,
			"snow3h":      0.0,
			"sprintf":     fmt.Sprintf,
		}
		return &env
	}

	env := map[string]interface{}{
		"temperature": data.CurrentTemp,
		"feelsLike":   data.FeelsLike,
		"rain1h":      data.Rain1h,
		"rain3h":      data.Rain3h,
		"snow1h":      data.Snow1h,
		"snow3h":      data.Snow3h,
		"sprintf":     fmt.Sprintf,
	}
	return &env
}

func compileMessage(message *Message, env map[string]interface{}) error {

	if message.Condition != "" {
		compiledCondition, err := expr.Compile(message.Condition, expr.Env(env))
		if err != nil {
			return err
		}
		message.compiledCondition = compiledCondition
	}

	variableNames := make(map[string]interface{})
	if message.Variables != nil {
		for i, v := range message.Variables {
			variableNames[v.Name] = ""
			if v.Choices != nil {
				for j, c := range v.Choices {
					program, err := expr.Compile(c.Expression, expr.Env(env))
					if err != nil {
						return err
					}
					message.Variables[i].Choices[j].program = program
				}
			}
		}
	}

	program, err := expr.Compile(message.Message, expr.Env(variableNames))
	if err != nil {
		return err
	}
	message.compiledMessage = program

	return nil
}

func evaluateMessage(message *Message, env map[string]interface{}) (string, error) {
	log.Debug("Evaluating message: " + message.Message)
	// If we have a condition, evaluate that first
	if message.compiledCondition != nil {
		output, err := expr.Run(message.compiledCondition, env)
		if err != nil {
			return "", err
		}
		result, ok := output.(bool)
		if !ok {
			return "", errors.New("Condition didn't evaluate to boolean")
		}
		if !result {
			return "", nil
		}
	}

	// Evaluate all variables
	setEnvironment := map[string]interface{}{}
	for _, v := range message.Variables {
		value := evaluateVariable(&v, &env)
		setEnvironment[v.Name] = value
	}

	// Evaluate the final expression
	output, err := expr.Run(message.compiledMessage, setEnvironment)
	if err != nil {
		return "", err
	}
	// Make sure we actually got a string
	result, ok := output.(string)
	if ok {
		return result, nil
	}
	return "", errors.New("Expression did not return a valid string")
}

// evaluateVariable returns a choice for a given variable. If the variable
// has no choices, an empty string is returned. If none of the variables
// evaluate, '<>' is returned.
func evaluateVariable(v *Variable, env *map[string]interface{}) string {
	log.Debug("Evaluating " + v.Name)
	if v.Choices == nil {
		log.Debug("Variable doesn't have any choices - will always return ''")
		return ""
	}

	for _, c := range v.Choices {
		output, err := expr.Run(c.program, *env)
		if err != nil {
			log.Error("Error evaluating choice ", err)
			continue
		}
		result, ok := output.(bool)
		if ok && result {
			log.Debug("Evaluated to: ", c.Value)
			return c.Value
		}
	}

	return "<>"
}

func Compile(messages *[]Message) error {
	env := buildEnv(nil)

	for i, _ := range *messages {
		err := compileMessage(&((*messages)[i]), *env)
		if err != nil {
			return err
		}
	}

	return nil
}

func Evaluate(data *owm_handler.EvaluationData, messages *[]Message) []string {
	processedMessages := []string{}
	env := buildEnv(data)

	for i := range *messages {
		output, err := evaluateMessage(&((*messages)[i]), *env)
		if err != nil {
			log.Error("Could not evaluate message: ", err)
		}
		processedMessages = append(processedMessages, output)
	}

	return processedMessages
}
