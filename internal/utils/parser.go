package utils

import (
	"encoding/json"
	"regexp"
	"strings"

	"gorm.io/datatypes"
)

// input is html text with variables in the form of {{variable}}
// output is a map of variables and their values
func ParseVariables(html string) (map[string]string, error) {
	variables := make(map[string]string)

	re := regexp.MustCompile(`{{\s*(\w+)\s*}}`)
	matches := re.FindAllStringSubmatch(html, -1)

	for _, match := range matches {
		variables[match[1]] = match[1]
	}

	return variables, nil
}

// input is a string with variables in the form of {{variable}}
// output is a string with the variables replaced by their values
func ReplaceVariables(input string, variables map[string]string) string {
	for variable, value := range variables {
		input = strings.Replace(input, "{{"+variable+"}}", value, -1)
	}
	return input
}

// convert datatypes.JSON to map[string]string
func JSONToMap(jsonData datatypes.JSON) (map[string]string, error) {
	var result map[string]string
	err := json.Unmarshal(jsonData, &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// usecase is to replace all the images in the html with our redirect url
// so we can track the number of opens
func ReplaceImagesWithRedirect(html string) string {
	re := regexp.MustCompile(`<img src="([^"]+)"`)
	html = re.ReplaceAllString(html, `<img src="https://kori.so/img/$1">`)
	return html
}
