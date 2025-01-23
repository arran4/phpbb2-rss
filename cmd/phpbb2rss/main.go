package main

import (
	"flag"
	"fmt"
	"github.com/arran4/phpbb2-rss"
	"io"
	"log"
	"os"
)

func main() {
	var forumURL string
	var out io.Writer = os.Stdout

	setOutputFile := func(filename string) error {
		if closer, ok := out.(io.Closer); ok {
			_ = closer.Close()
		}
		file, err := os.Create(filename)
		if err != nil {
			return err
		}
		out = file
		return nil
	}

	flag.StringVar(&forumURL, "url", "", "URL of the PHPBB2 forum 24-hour page")
	flag.Func("output", "Output file for the RSS feed", setOutputFile)
	flag.Parse()

	if forumURL == "" {
		log.Fatal("A forum URL must be provided using the -url flag.")
	}

	rss, err := phpbb2rss.FetchAndGenerateRSS(forumURL)
	if err != nil {
		log.Fatalf("Error generating RSS feed: %v", err)
	}

	_, err = fmt.Fprintf(out, "%s", rss)
	if err != nil {
		log.Fatalf("Error writing RSS feed: %v", err)
	}
}
