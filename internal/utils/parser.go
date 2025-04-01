package utils

import (
	"encoding/json"
	"fmt"
	"kori/internal/config"
	"kori/internal/utils/base64"
	"kori/internal/utils/logger"
	"regexp"
	"strings"

	"github.com/golang-jwt/jwt"
	"gorm.io/datatypes"
)

var console = logger.New("console")

// ParseVariables input is html text with variables in the form of {{variable}} or {{ variable.subvariable }} or {{ varible }}
// output is a map of variables and their values
// input: <p>Hello, {{name}}! {{name.first}} {{name.last}}</p>
// output: map[name:John, name.first:John, name.last:Doe]
func ParseVariables(html string) (map[string]string, error) {
	variables := make(map[string]string)

	re := regexp.MustCompile(`{{\s*(\w+(?:\.\w+)*)\s*}}`)
	matches := re.FindAllStringSubmatch(html, -1)

	for _, match := range matches {
		variables[match[1]] = match[1]
	}

	return variables, nil
}

// ReplaceVariables input is html text with variables in the form of {{variable}} or {{ variable.subvariable }} or {{ varible }}
// output is a string with the variables replaced by their values
func ReplaceVariables(input string, variables map[string]string, mailId string, cfg *config.Config, trackLinks bool) string {
	for variable, value := range variables {
		re := regexp.MustCompile(`{{\s*` + regexp.QuoteMeta(variable) + `(?:\.\w+)*\s*}}`)
		input = re.ReplaceAllString(input, value)
	}
	if trackLinks {
		input = ReplaceLinksWithRedirect(input, mailId, cfg)
	}

	return base64.EncodeToBase64(input)
}

// JSONToMap convert datatypes.JSON to map[string]string
func JSONToMap(jsonData datatypes.JSON) (map[string]string, error) {
	var result map[string]string
	err := json.Unmarshal(jsonData, &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// MapToJSON convert map[string]string to datatypes.JSON
func MapToJSON(data map[string]string) (datatypes.JSON, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	return jsonData, nil
}

// ReplaceLinksWithRedirect usecase is to replace all the links in the html with our redirect url
// so we can track the number of clicks
func ReplaceLinksWithRedirect(html string, mailId string, cfg *config.Config) string {
	// Replace anchor href links with tracking URL to track clicks
	hrefRe := regexp.MustCompile(`<a[^>]+href="([^"]+)"`)

	// hash mailId into jwt
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"mailId": mailId,
	})
	tokenString, err := token.SignedString([]byte(cfg.JWT.Secret))
	if err != nil {
		console.Error("Error signing token: %v", err)
		return html
	}

	html = hrefRe.ReplaceAllString(html, fmt.Sprintf(`<a href="%s/track/click/$1?token=%s"`, cfg.Server.PublicURL, tokenString))

	// Add tracking pixel at bottom of email to track opens
	html = html + fmt.Sprintf(`<img src="%s/track/open?token=%s" style="display:none" width="1" height="1">`, cfg.Server.PublicURL, tokenString)

	// add unsubcribe link to the input this needs to go before the closing body tag
	html = strings.Replace(html, "</body>", fmt.Sprintf(`<table><tr><td><a href="%s/track/unsubscribe?token=%s">Unsubscribe from this list</a></td></tr></table></body>`, cfg.Server.PublicURL, tokenString), 1)

	return html
}
