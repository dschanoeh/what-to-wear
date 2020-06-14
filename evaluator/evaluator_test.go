package evaluator

import "testing"

func TestEvaluateRule(t *testing.T) {
	set := Message{
		Message: "'Test is ' + test",
		Variables: []Variable{
			{
				Name: "test",
				Choices: []Choice{
					{
						Expression: `temperature < 20 && temperature > 10`,
						Value:      "test",
					},
					{
						Expression: `temperature <= 10`,
						Value:      "foo",
					},
				},
			},
		},
	}

	env := buildEnv(nil)
	(*env)["temperature"] = 15

	compileMessage(&set, *env)
	s, err := evaluateMessage(&set, *env)
	if err != nil {
		t.Error("An error was returned: ", err)
	}
	if s != "Test is test" {
		t.Error("Result is: ", s)
	}
}

func TestEvaluateRule2(t *testing.T) {
	set := Message{
		Message:   "'Bring an umbrella'",
		Condition: `temperature < 20`,
	}

	env := buildEnv(nil)
	(*env)["temperature"] = 15

	compileMessage(&set, *env)
	s, err := evaluateMessage(&set, *env)
	if err != nil {
		t.Error("An error was returned: ", err)
	}
	if s != "Bring an umbrella" {
		t.Error("Result is: ", s)
	}
}

func TestEvaluateRule3(t *testing.T) {
	set := Message{
		Message:   "'Bring an umbrella'",
		Condition: `temperature < 20`,
	}

	env := buildEnv(nil)
	(*env)["temperature"] = 21

	compileMessage(&set, *env)
	s, err := evaluateMessage(&set, *env)
	if err != nil {
		t.Error("An error was returned: ", err)
	}
	if s != "" {
		t.Error("Result is: ", s)
	}
}

func TestNegativeMessage(t *testing.T) {
	set := Message{
		Message:         "'Bring an umbrella'",
		NegativeMessage: "'Bring two umbrellas'",
		Condition:       `temperature < 20`,
	}

	env := buildEnv(nil)
	(*env)["temperature"] = 21

	compileMessage(&set, *env)
	s, err := evaluateMessage(&set, *env)
	if err != nil {
		t.Error("An error was returned: ", err)
	}
	if s != "Bring two umbrellas" {
		t.Error("Result is: ", s)
	}
}
