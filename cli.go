package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"strconv"
)

func startCLI() {

	var initial_url string
	flag.StringVar(&initial_url, "start", "zachmanson.com", "starting url")
	flag.Parse()

	fmt.Println("Starting crawler from:", initial_url)

	CrawlDomain(initial_url)

	db, _ := sql.Open("sqlite3", "./crawler.db?_busy_timeout=100000") // Open the created SQLite File
	if db == nil {
		log.Fatal("db nil")
	}
	defer db.Close() // Defer Closing the database

	CmdLoop(db)

}

func CmdLoop(db *sql.DB) {
	var input string
	var domains []string

	for {
		fmt.Println("Enter 'exit' to exit")
		domains = GetNewDomains()

		for i, domain := range domains {
			fmt.Println(i+1, ":", domain)
		}

		// fmt.Println("New domains:", domains)
		fmt.Scan(&input)

		if input == "exit" {
			fmt.Printf("Exiting...\n")
			break
		}

		// cast input to int
		num, err := strconv.Atoi(input)
		if err != nil {
			continue
		}
		// if input is negative number, blacklist domain
		if num < 0 {
			RejectDomain(domains[(-num)-1])
		} else {
			go CrawlDomain(domains[num-1])
		}
	}
}
