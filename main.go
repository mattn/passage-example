package main

import (
	"embed"
	"html/template"
	"io"
	"net/http"
	"os"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/passageidentity/passage-go"
)

var (
	passageAppID  = os.Getenv("PASSAGE_APP_ID")
	passageApiKey = os.Getenv("PASSAGE_API_KEY")

	//go:embed static
	assets embed.FS
)

func authRequired(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		psg, err := passage.New(passageAppID, &passage.Config{
			APIKey: passageApiKey,
		})
		if err != nil {
			// This should not fail, but abort the request if it does
			c.Echo().Logger.Debug(err)
			return c.NoContent(http.StatusInternalServerError)
		}
		passageUserID, err := psg.AuthenticateRequest(c.Request())
		if err != nil {
			// Authentication failed!
			return c.Render(http.StatusOK, "unauthorized", nil)
		}

		// Get the authenticated user's email and set it in the context
		passageUser, err := psg.GetUser(passageUserID)
		if err != nil {
			// This should not fail, but abort the request if it does
			c.Echo().Logger.Debug(err)
			return c.String(http.StatusInternalServerError, err.Error())
		}
		c.Set("userEmail", passageUser.Email)

		// Authentication was successful, proceed.
		return next(c)
	}
}

type TemplateRender struct {
	templates *template.Template
}

func (t *TemplateRender) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

func main() {
	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.Gzip())
	e.Debug = true

	renderer := &TemplateRender{
		templates: template.Must(template.ParseFS(assets, "static/*.tmpl")),
	}
	e.Renderer = renderer

	e.GET("/", func(c echo.Context) error {
		return c.Render(http.StatusOK, "index", map[string]any{
			"AppID": passageAppID,
		})
	})

	group := e.Group("", authRequired)
	group.GET("/dashboard", func(c echo.Context) error {
		return c.Render(http.StatusOK, "dashboard", map[string]any{
			"email": c.Get("userEmail").(string),
		})
	})

	e.Logger.Fatal(e.Start(":8080"))
}
