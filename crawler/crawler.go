package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gocolly/colly"
)

func getHostname(link string) string {
	urlObj, err := url.Parse(link)
	if err != nil {
		log.Println(err)
		return ""
	}
	hostname := strings.TrimPrefix(urlObj.Hostname(), "www.")
	return hostname
}

func ResetDatabase() {
	db, _ := sql.Open("sqlite3", "./crawler.db?_busy_timeout="+strconv.Itoa(TIMEOUT))
	if db == nil {
		log.Fatal("db nil")
	}
	defer db.Close() // Defer Closing the database

	_, err := db.Exec("DROP TABLE IF EXISTS link")
	if err != nil {
		log.Fatalln(err.Error())
	}
	_, err = db.Exec("DROP TABLE IF EXISTS domain")
	if err != nil {
		log.Fatalln(err.Error())
	}
	_, err = db.Exec(
		`CREATE TABLE link (
				id INTEGER PRIMARY KEY,
				"start_domain" TEXT,
				"end_domain" TEXT,
				"start_url" TEXT,
				"end_url" TEXT,
				UNIQUE("start_url", "end_url")
		)`,
	)
	if err != nil {
		log.Fatalln(err.Error())
	}
	_, err = db.Exec(
		`CREATE TABLE domain (
			"domain" TEXT PRIMARY KEY,
			"valid" INTEGER
		)`,
	)
	if err != nil {
		log.Fatalln(err.Error())
	}
}

func RejectDomain(domain string) {
	db, _ := sql.Open("sqlite3", "./crawler.db?_busy_timeout="+strconv.Itoa(TIMEOUT))
	if db == nil {
		log.Fatal("db nil")
	}
	defer db.Close() // Defer Closing the database
	fmt.Println("REJECTED:", domain)
	_, err := db.Exec("UPDATE domain SET valid = -1 WHERE domain = ?", domain)
	if err != nil {
		log.Fatalln(err.Error())
	}
}

func startedDomain(domain string) {
	db, _ := sql.Open("sqlite3", "./crawler.db?_busy_timeout="+strconv.Itoa(TIMEOUT))
	if db == nil {
		log.Fatal("db nil")
	}
	defer db.Close() // Defer Closing the database
	log.Println("STARTED:", domain)
	_, err := db.Exec("UPDATE domain SET valid = 1 WHERE domain = ?", domain)
	if err != nil {
		log.Fatalln(err.Error())
	}
}

func completeDomain(domain string) {
	db, _ := sql.Open("sqlite3", "./crawler.db?_busy_timeout="+strconv.Itoa(TIMEOUT))
	if db == nil {
		log.Fatal("db nil")
	}
	defer db.Close() // Defer Closing the database

	log.Println("COMPLETED:", domain)
	_, err := db.Exec("UPDATE domain SET valid = 2 WHERE domain = ?", domain)
	if err != nil {
		log.Fatalln(err.Error())
	}
}

func GetNewDomains() []string {
	db, _ := sql.Open("sqlite3", "./crawler.db?_busy_timeout="+strconv.Itoa(TIMEOUT))
	if db == nil {
		log.Fatal("db nil")
	}
	defer db.Close() // Defer Closing the database

	row, err := db.Query("SELECT * FROM domain WHERE valid = 0 LIMIT 10")
	if err != nil {
		log.Fatal(err)
	}
	defer row.Close()
	var domains []string
	for row.Next() { // Iterate and fetch the records from result cursor
		var domain string
		var valid string
		row.Scan(&domain, &valid)
		domains = append(domains, domain)
	}
	return domains
}

func insertLink(start_url string, end_url string) {
	db, _ := sql.Open("sqlite3", "./crawler.db?_busy_timeout="+strconv.Itoa(TIMEOUT))
	if db == nil {
		log.Fatal("db nil")
	}
	defer db.Close() // Defer Closing the database

	if VERBOSE {
		log.Println("Inserting link:\n", start_url, "\n", end_url)
	}
	insertSQL := `INSERT INTO link(start_domain, end_domain, start_url, end_url) VALUES (?, ?, ?, ?)`
	statement, err := db.Prepare(insertSQL)
	defer statement.Close()
	if err != nil {
		log.Fatalln(err.Error())
	}
	_, err = statement.Exec(getHostname(start_url), getHostname(end_url), start_url, end_url)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			if VERBOSE {
				log.Println("Domain already exists")
			}
			return
		} else {
			log.Fatalln(err.Error())
		}
	}

	// attempt to insert domain

	insertSQL = `INSERT INTO domain(domain, valid) VALUES (?, ?)`
	statement, err = db.Prepare(insertSQL)
	if err != nil {
		log.Fatalln(err.Error())
	}
	_, err = statement.Exec(getHostname(end_url), 0)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			if VERBOSE {
				log.Println("Domain already exists")
			}
		} else {
			log.Fatalln(err.Error())
		}
	}
}

func CrawlDomain(url string) {

	startedDomain(url)
	domain := getHostname(url)

	// only crawl a single domain at a time
	c := colly.NewCollector(
		colly.AllowedDomains(url, "www."+url),
		colly.Async(true),
	)

	c.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Parallelism: 2,
		RandomDelay: 5 * time.Second,
	})

	c.OnHTML("a", func(e *colly.HTMLElement) {
		pageUrl := e.Request.URL.String()
		link := e.Attr("href")
		if VERBOSE {

			fmt.Println("Found link:", link)
		}

		if getHostname(link) != domain {
			insertLink(pageUrl, link)
		}

		e.Request.Visit(link)
	})

	c.OnRequest(func(r *colly.Request) {
		if VERBOSE {
			fmt.Println("Visiting", r.URL.String())
		}
	})

	c.OnError(func(r *colly.Response, err error) {
		log.Println("Something went wrong:", err, r.Request.URL)
	})

	c.OnScraped(func(r *colly.Response) {
		if VERBOSE {
			fmt.Println("Finished", r.Request.URL)
		}
	})

	c.Visit("https://" + url)
	c.Wait()

	completeDomain(url)
}
