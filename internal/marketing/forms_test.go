package marketing_test

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
	"kori/internal/handlers"
	"kori/internal/models"
)

func TestFormBrandingAndHTMLAction(t *testing.T) {
	db := testDB(t)
	team, list := uuid.NewString(), uuid.NewString()
	seed(t, db, &models.MailingList{Base: models.Base{ID: list}, TeamID: team, Name: "Audience"})
	h := handlers.NewMarketingHandler(db, strings.Repeat("s", 32), "https://api.example.com")
	e := echo.New()
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error { c.Set("teamID", team); return next(c) }
	})
	e.POST("/forms", h.SaveForm)
	e.PUT("/forms/:id", h.SaveForm)
	e.GET("/forms", h.Forms)
	e.GET("/public/:slug", h.PublicForm)
	e.POST("/public/:slug", h.SubmitForm)
	call := func(method, path, contentType, body string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		r.Header.Set("Content-Type", contentType)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, r)
		return rec
	}
	input := map[string]interface{}{"name": "Custom signup", "listId": list, "status": "PUBLISHED", "successMessage": "Welcome <script>alert(1)</script>", "fields": []map[string]interface{}{{"label": "Email", "type": "EMAIL", "key": "email", "required": true}}, "theme": map[string]string{"preset": "dark", "backgroundColor": "#15151c", "cardColor": "#242430", "textColor": "#f4f2ff", "buttonColor": "#c4b5fd", "buttonTextColor": "#211736", "logoUrl": "https://example.com/logo.png", "font": "serif", "corners": "square"}}
	raw, _ := json.Marshal(input)
	rec := call("POST", "/forms", echo.MIMEApplicationJSON, string(raw))
	require.Equal(t, 200, rec.Code, rec.Body.String())
	var form models.Form
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &form))
	rec = call("GET", "/public/"+form.Slug, "", "")
	require.Equal(t, 200, rec.Code)
	var public map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &public))
	require.Equal(t, "#15151c", public["theme"].(map[string]interface{})["backgroundColor"])
	require.NotContains(t, public, "TeamID")
	// Older clients omitting theme must preserve customization.
	delete(input, "theme")
	raw, _ = json.Marshal(input)
	rec = call("PUT", "/forms/"+form.ID, echo.MIMEApplicationJSON, string(raw))
	require.Equal(t, 200, rec.Code, rec.Body.String())
	require.Contains(t, rec.Body.String(), "https://example.com/logo.png")
	// Server rejects script URLs and CSS payloads.
	for _, theme := range []map[string]string{{"logoUrl": "javascript:alert(1)"}, {"buttonColor": "red;display:none"}, {"font": "url(evil)"}} {
		input["theme"] = theme
		raw, _ = json.Marshal(input)
		rec = call("PUT", "/forms/"+form.ID, echo.MIMEApplicationJSON, string(raw))
		require.Equal(t, 400, rec.Code)
	}
	path := "/public/" + form.Slug
	for _, payload := range []url.Values{{"email": {"reader@example.com"}}, {"email": {"invalid"}, "consent": {"true"}}, {"email": {""}, "consent": {"on"}}} {
		rec = call("POST", path, echo.MIMEApplicationForm, payload.Encode())
		require.Equal(t, 400, rec.Code, rec.Body.String())
		require.Contains(t, rec.Header().Get("Content-Type"), "text/html")
	}
	// Query values must not supply consent.
	rec = call("POST", path+"?consent=true", echo.MIMEApplicationForm, "email=reader%40example.com")
	require.Equal(t, 400, rec.Code)
	payload := url.Values{"email": {"reader@example.com"}, "consent": {"on"}, "requestId": {uuid.NewString()}}
	for i := 0; i < 2; i++ {
		rec = call("POST", path, echo.MIMEApplicationForm, payload.Encode())
		require.Equal(t, 200, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), "&lt;script&gt;")
		require.NotContains(t, rec.Body.String(), "<script>")
	}
	var count int64
	require.NoError(t, db.Model(&models.FormSubmission{}).Count(&count).Error)
	require.EqualValues(t, 1, count)
	// Plain HTML needs no generated identifier, multipart works too.
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	require.NoError(t, writer.WriteField("email", "second@example.com"))
	require.NoError(t, writer.WriteField("consent", "true"))
	require.NoError(t, writer.Close())
	rec = call("POST", path, writer.FormDataContentType(), body.String())
	require.Equal(t, 200, rec.Code, rec.Body.String())
	var contact models.Contact
	require.NoError(t, db.Where("email = ?", "second@example.com").First(&contact).Error)
	require.Equal(t, list, contact.ListID)
	require.Equal(t, team, contact.TeamID)
	rec = call("POST", path, echo.MIMEApplicationForm, "email=bot%40example.com&consent=on&website=spam")
	require.Equal(t, 200, rec.Code)
	require.NoError(t, db.Model(&models.FormSubmission{}).Count(&count).Error)
	require.EqualValues(t, 2, count)
	require.NoError(t, db.Model(&form).Update("status", "DRAFT").Error)
	rec = call("POST", path, echo.MIMEApplicationForm, "email=third%40example.com&consent=on")
	require.Equal(t, 404, rec.Code)
}
