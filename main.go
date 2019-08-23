package main

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"os"

	"github.com/BurntSushi/toml"
	"github.com/gocolly/colly"
	log "github.com/withmandala/go-log"
)

type OracleConfig struct {
	UserAgent string
	CacheDir  string
	IgnoreTLS bool
	Debug     bool
}

func newResponseLogger(logger *log.Logger) func(r *colly.Response) {

	return func(r *colly.Response) {
		logger.Debugf("Response from %s: Status <%d>\n", r.Request.URL, r.StatusCode)
	}
}

func newErrorLogger(logger *log.Logger) func(r *colly.Response, err error) {
	return func(r *colly.Response, err error) {
		logger.Warnf("Something went wrong: %s", err)
	}
}

func main() {
	fmt.Println("Hello world")

	logger := log.New(os.Stdout).WithColor()

	config := OracleConfig{UserAgent: "Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:68.0) Gecko/20100101 Firefox/68.0", CacheDir: "cache"}

	_, err := toml.DecodeFile("config.toml", &config)

	if err != nil {
		logger.Errorf("Could not parse config file due to error: %s\n", err)
		return
	}

	if config.Debug {
		logger = logger.WithDebug()
	}

	var articleCollector *colly.Collector = colly.NewCollector()
	var paperCollector *colly.Collector

	articleCollector.Async = true

	if config.IgnoreTLS {
		tr := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
		articleCollector.WithTransport(tr)
	}

	articleCollector.CacheDir = config.CacheDir
	articleCollector.UserAgent = config.UserAgent

	paperCollector = articleCollector.Clone()

	articleCollector.OnResponse(newResponseLogger(logger))
	paperCollector.OnResponse(newResponseLogger(logger))

	articleCollector.OnError(newErrorLogger(logger))
	paperCollector.OnError(newErrorLogger(logger))

	articleCollector.OnRequest(func(r *colly.Request) {
		logger.Debugf("Visiting %s", r.URL)
	})

	articleCollector.OnXML("//item", func(item *colly.XMLElement) {

		logger.Infof("found item %s\n", item.ChildText("./title"))
		articleCollector.Visit(item.ChildText("./link"))

	})

	articleCollector.OnHTML("#wrapper", func(el *colly.HTMLElement) {

		logger.Infof("News article %s\n", el.ChildText(".article h1.page_title"))

		//links = el.

		el.ForEach("#sidebar-content a[rel='nofollow']", func(i int, link *colly.HTMLElement) {
			fmt.Printf("Link: %s\n", link.Attr("href"))

			ctx := link.Request.Ctx
			ctx.Put("LinkedNewsArticle", link.Request.URL.String())

			paperCollector.Request("GET", link.Attr("href"), nil, ctx, nil)

		})
	})

	paperCollector.OnHTML("meta", func(el *colly.HTMLElement) {

		if el.Attr("name") == "dc.Identifier" || el.Attr("name") == "DOI" {
			newsURL := el.Request.Ctx.Get("LinkedNewsArticle")

			logger.Infof("Found doi %s associated with %s", el.Attr("content"), newsURL)
		}

	})

	//c.Visit("https://www.eurekalert.org/rss/technology_engineering.xml")

	articleCollector.Visit("https://www.eurekalert.org/pub_releases/2019-08/fm-taw081619.php")

	articleCollector.Wait()
	paperCollector.Wait()
}
