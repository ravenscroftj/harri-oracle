package main

import (
	"os"

	"github.com/BurntSushi/toml"
	"github.com/ravenscroftj/harri-oracle/oracle"
	log "github.com/withmandala/go-log"
)

func main() {

	logger := log.New(os.Stdout).WithColor()

	config := oracle.OracleConfig{UserAgent: "Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:68.0) Gecko/20100101 Firefox/68.0", CacheDir: "cache"}

	_, err := toml.DecodeFile("config.toml", &config)

	if err != nil {
		logger.Errorf("Could not parse config file due to error: %s\n", err)
		return
	}

	if config.Debug {
		logger = logger.WithDebug()
	}

	logger.Infof("Initialising scraper")

	//c.Visit()

	//articleCollector.Visit("https://www.eurekalert.org/pub_releases/2019-08/fm-taw081619.php")

	//articleCollector.Wait()
	//paperCollector.Wait()

	scraper := oracle.NewScraper(config, logger)
	scraper.AddFeed("https://www.eurekalert.org/rss/technology_engineering.xml")

	go scraper.Run()

	scraper.Await()

}
