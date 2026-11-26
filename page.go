package main

import (
	"fmt"
	"log"
	"net/http"
	"runtime"
	"time"

	g "maragu.dev/gomponents"
	hx "maragu.dev/gomponents-htmx"
	c "maragu.dev/gomponents/components"
	. "maragu.dev/gomponents/html"
)

const (
	deployURL = "https://dollardeploy.com/r/go-boilerplate"
	docsURL   = "https://docs.dollardeploy.com"
)

func render(w http.ResponseWriter, n g.Node) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := n.Render(w); err != nil {
		log.Printf("render: %v", err)
	}
}

// lucide icons (lucide.dev), inlined so the starter needs no icon package.
var icons = map[string]string{
	"GitBranch":   `<line x1="6" x2="6" y1="3" y2="15"/><circle cx="18" cy="6" r="3"/><circle cx="6" cy="18" r="3"/><path d="M18 9a9 9 0 0 1-9 9"/>`,
	"ShieldCheck": `<path d="M20 13c0 5-3.5 7.5-7.66 8.95a1 1 0 0 1-.67-.01C7.5 20.5 4 18 4 13V6a1 1 0 0 1 1-1c2 0 4.5-1.2 6.24-2.72a1.17 1.17 0 0 1 1.52 0C14.51 3.81 17 5 19 5a1 1 0 0 1 1 1z"/><path d="m9 12 2 2 4-4"/>`,
	"Clock":       `<circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/>`,
	"Activity":    `<path d="M22 12h-2.48a2 2 0 0 0-1.93 1.46l-2.35 8.36a.25.25 0 0 1-.48 0L9.24 2.18a.25.25 0 0 0-.48 0l-2.35 8.36A2 2 0 0 1 4.49 12H2"/>`,
}

var features = [][3]string{
	{"GitBranch", "Deploys from GitHub", "Every push to main builds and ships on your server."},
	{"ShieldCheck", "HTTPS", "Certificates issued and renewed for every domain."},
	{"Clock", "Cron", "Scheduled calls to /api/cron, with run history and logs."},
	{"Activity", "Monitoring and logs", "Host metrics, app logs and backups, built in."},
}

var steps = [][2]string{
	{"01", "git push origin main"},
	{"02", "go build -o app ."},
	{"03", "live at https://app.example.com"},
}

func icon(name string) g.Node {
	return g.Raw(`<svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">` + icons[name] + `</svg>`)
}

func logo() g.Node {
	return g.Raw(`<svg class="logo" viewBox="0 0 123 122" fill="currentColor" aria-hidden="true"><path d="M24.68.2h97.55v97.55L97.98 73.49V24.45H48.93z"/><path d="M48.93 24.45 97.98 73.5V122L.43 24.45z"/></svg>`)
}

// serverFacts is rendered by this Go process and re-fetched by htmx every 2 seconds.
func serverFacts() g.Node {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	rows := [][2]string{
		{"runtime", runtime.Version()},
		{"platform", runtime.GOOS + "/" + runtime.GOARCH},
		{"uptime", time.Since(startedAt).Truncate(time.Second).String()},
		{"goroutines", fmt.Sprint(runtime.NumGoroutine())},
		{"memory", fmt.Sprintf("%d MB", m.Sys/1048576)},
		{"rendered", time.Now().UTC().Format("15:04:05") + " UTC"},
	}
	return Dl(Class("facts"),
		hx.Get("/fragments/facts"), hx.Trigger("every 2s"), hx.Swap("outerHTML"),
		g.Map(rows, func(r [2]string) g.Node {
			return g.Group{Dt(g.Text(r[0])), Dd(g.Text(r[1]))}
		}),
	)
}

func homePage() g.Node {
	return c.HTML5(c.HTML5Props{
		Title:       "Go Boilerplate · DollarDeploy",
		Description: "A minimal Go starter, deployed to a server you own with DollarDeploy.",
		Language:    "en",
		Head: g.Group{
			Link(Rel("icon"), Href("/favicon.ico")),
			Link(Rel("preconnect"), Href("https://fonts.googleapis.com")),
			Link(Rel("preconnect"), Href("https://fonts.gstatic.com"), g.Attr("crossorigin")),
			Link(Rel("stylesheet"), Href("https://fonts.googleapis.com/css2?family=Inter:wght@400;500&family=JetBrains+Mono&display=swap")),
			Link(Rel("stylesheet"), Href("/static/app.css")),
			Script(Src("/static/htmx.min.js"), Defer()),
		},
		Body: g.Group{
			Div(Class("page"),
				Header(Class("top"),
					A(Href(deployURL), Class("brand"), logo(), g.Text("DollarDeploy")),
					A(Href(docsURL), Class("mono muted link"), g.Text("docs")),
				),
				Main(
					Section(Class("hero"),
						Div(
							P(Class("eyebrow"), g.Text("Go boilerplate")),
							H1(
								Span(Class("h1-top"), g.Text("Your app. Your server.")),
								Span(Class("h1-dither"), g.Text("Zero DevOps.")),
							),
							P(Class("lede"), g.Text("Ship this starter from GitHub to a server you own, with HTTPS, cron and redeploys on every push.")),
							Div(Class("ctas"),
								A(Href(deployURL), Class("btn btn-primary"), g.Text("Deploy this template")),
								A(Href(docsURL), Class("btn"), g.Text("Read the docs")),
							),
						),
						Div(Class("launch"),
							g.El("picture",
								Source(g.Attr("media", "(prefers-color-scheme: dark)"), g.Attr("srcset", "/launch-dark.gif")),
								Img(Src("/launch-light.gif"), Width("160"), Height("96"),
									Alt("Dithered pixel animation of a rocket launching next to two servers")),
							),
						),
					),
					Section(Class("split"),
						Div(
							H2(g.Text("From push to production.")),
							Ol(Class("steps"),
								g.Map(steps, func(s [2]string) g.Node {
									return Li(Span(Class("muted"), g.Text(s[0])), g.Text(s[1]))
								}),
							),
						),
						Div(
							H2(g.Text("Rendered on your server.")),
							P(Class("sub"), g.Text("Live from this Go process, refreshed by htmx every 2 seconds.")),
							serverFacts(),
						),
					),
					Section(Class("features"),
						g.Map(features, func(f [3]string) g.Node {
							return Div(Class("feature"), icon(f[0]), H3(g.Text(f[1])), P(g.Text(f[2])))
						}),
					),
				),
				Footer(Class("foot"),
					Span(g.Text("edit page.go to start")),
					A(Href(deployURL), Class("link"), g.Text("dollardeploy.com")),
				),
			),
		},
	})
}
