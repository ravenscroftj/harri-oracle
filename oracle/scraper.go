package oracle

import (
	"crypto/tls"
	"net/http"
	"time"

	"github.com/gocolly/colly"
	log "github.com/withmandala/go-log"
)

type Scraper struct {
	feeds            []string
	Interval         int
	stopChan         chan bool
	finishedChan     chan bool
	logger           *log.Logger
	articleCollector *colly.Collector
	paperCollector   *colly.Collector
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

func NewScraper(config OracleConfig, logger *log.Logger) Scraper {

	s := Scraper{}
	s.feeds = make([]string, 10)
	s.Interval = 10
	s.stopChan = make(chan bool, 1)
	s.finishedChan = make(chan bool, 1)
	s.logger = logger

	c := colly.NewCollector()
	c.Async = true
	c.UserAgent = config.UserAgent
	c.CacheDir = config.CacheDir

	if config.IgnoreTLS {
		tr := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
		c.WithTransport(tr)
	}

	// generate collectors based on common config
	s.articleCollector = c.Clone()
	s.paperCollector = c.Clone()

	// register standard logger handlers for collector events
	s.articleCollector.OnResponse(newResponseLogger(logger))
	s.paperCollector.OnResponse(newResponseLogger(logger))
	s.articleCollector.OnError(newErrorLogger(logger))
	s.paperCollector.OnError(newErrorLogger(logger))

	// register scraper specific handlers
	s.buildArticleCollector()
	s.buildPaperCollector()

	return s
}

/* Register logic for scraping scientific paper information */
func (s *Scraper) buildPaperCollector() {

	s.paperCollector.OnHTML("meta", func(el *colly.HTMLElement) {

		if el.Attr("name") == "dc.Identifier" || el.Attr("name") == "DOI" {
			newsURL := el.Request.Ctx.Get("LinkedNewsArticle")

			s.logger.Infof("Found doi %s associated with %s", el.Attr("content"), newsURL)
		}

	})
}

/* Register logic for scraping news articles linked to scientific papers */
func (s *Scraper) buildArticleCollector() {

	s.articleCollector.OnXML("//item", func(item *colly.XMLElement) {

		s.logger.Infof("found item %s\n", item.ChildText("./title"))
		s.articleCollector.Visit(item.ChildText("./link"))

	})

	s.articleCollector.OnHTML("#wrapper", func(el *colly.HTMLElement) {

		s.logger.Infof("News article %s\n", el.ChildText(".article h1.page_title"))

		el.ForEach("#sidebar-content a[rel='nofollow']", func(i int, link *colly.HTMLElement) {

			ctx := link.Request.Ctx
			ctx.Put("LinkedNewsArticle", link.Request.URL.String())

			s.paperCollector.Request("GET", link.Attr("href"), nil, ctx, nil)

		})
	})
}

func (s *Scraper) AddFeed(feedURL string) {
	s.feeds = append(s.feeds, feedURL)
}

func (s *Scraper) Run() {

	stopLoop := false
	for !stopLoop {
		select {
		case <-s.stopChan:
			s.logger.Infof("Stopping crawler...\n")
			stopLoop = true
			break
		default:
			s.logger.Infof("doing scraper loop\n")
		}

		s.scrapeloop()

		if !stopLoop {
			time.Sleep(time.Duration(s.Interval) * time.Second)
		}

	}

	s.logger.Infof("Crawler stopped... notifying listeners...\n")
	s.finishedChan <- true
}

func (s *Scraper) Stop() {
	s.stopChan <- true
}

func (s *Scraper) Await() {
	<-s.finishedChan
}

/* Scrape loop runs the actual scrape logic */
func (s *Scraper) scrapeloop() {

	s.logger.Infof("Execute scraper loop")

	for _, url := range s.feeds {
		s.articleCollector.Visit(url)
	}

}
