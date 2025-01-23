package phpbb2rss

import (
	"encoding/xml"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type RSS struct {
	XMLName xml.Name `xml:"rss"`
	Version string   `xml:"version,attr"`
	Channel Channel  `xml:"channel"`
}

type Channel struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	Items       []Item `xml:"item"`
}

type Item struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
	GUID        string `xml:"guid"`
}

func FetchAndGenerateRSS(forumURL string) (string, error) {
	resp, err := http.Get(forumURL)
	if err != nil {
		return "", fmt.Errorf("failed to fetch forum page: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to parse page content: %w", err)
	}

	pageTitle := ""
	doc.Find("a.nav:contains('Forum Index')").Each(func(i int, s *goquery.Selection) {
		// Extract text from the link and trim "Forum Index" if present
		pageTitle = strings.TrimSpace(s.Text())
		if strings.HasSuffix(pageTitle, "Forum Index") {
			pageTitle = strings.TrimSuffix(pageTitle, " Forum Index")
		}
	})

	if pageTitle == "" {
		pageTitle = doc.Find("title").Text()
	}

	if pageTitle == "" {
		pageTitle = "PHPBB2 Forum Topics" // Fallback title if none found
	}

	baseURL, err := url.Parse(forumURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse base URL: %w", err)
	}

	var items []Item
	doc.Find(".forumline tr").Each(func(i int, s *goquery.Selection) {
		topic := s.Find(".topictitle a").First()
		title := strings.TrimSpace(topic.Text())
		topicLink, topicExists := topic.Attr("href")
		latestPostLink, latestPostExists := s.Find("a:contains('View latest post')").Attr("href")
		if !topicExists || title == "" {
			return
		}

		// Use url.Parse to ensure proper handling of relative URLs
		topicURL, err := baseURL.Parse(topicLink)
		if err != nil {
			return
		}

		link := topicURL.String() // Full URL for the topic

		// If the latest post exists, use that link instead
		if latestPostExists {
			latestPostURL, err := baseURL.Parse(latestPostLink)
			if err != nil {
				return
			}
			link = latestPostURL.String() // Full URL for the latest post
		}

		pubDateRaw := strings.TrimSpace(s.Find(".postdetails").Last().Text())
		parsedDate, err := time.Parse("Mon Jan 02, 2006 3:04 pm", pubDateRaw)
		if err != nil {
			parsedDate = time.Now()
		}

		replies := strings.TrimSpace(s.Find("td:nth-child(4) .postdetails").Text())
		posts := strings.TrimSpace(s.Find("td:nth-child(5) .postdetails").Text())
		author := strings.TrimSpace(s.Find(".name a").First().Text())
		lastCommenter := strings.TrimSpace(s.Find(".row2 a[href*='profile']").Last().Text())
		pages := parsePageCount(s.Find("span.gensmall").Text())
		category := strings.TrimSpace(s.Find(".forumlink").Text())

		description := fmt.Sprintf("Category: %s\nAuthor: %s\nLast Commenter: %s\nReplies: %s\nPosts: %s\nPages: %s", category, author, lastCommenter, replies, posts, pages)

		titleWithCategory := fmt.Sprintf("[%s] %s", category, title)

		items = append(items, Item{
			Title:       titleWithCategory,
			Link:        link,
			Description: description,
			PubDate:     parsedDate.Format(time.RFC1123),
			GUID:        link,
		})
	})

	rss := RSS{
		Version: "2.0",
		Channel: Channel{
			Title:       pageTitle, // Dynamically set page title
			Link:        forumURL,
			Description: "RSS feed for topics from a PHPBB2 forum page",
			Items:       items,
		},
	}

	xmlData, err := xml.MarshalIndent(rss, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal RSS feed: %w", err)
	}

	return fmt.Sprintf("%s%s", xml.Header, xmlData), nil
}

func parsePageCount(gensmallText string) string {
	if !strings.Contains(gensmallText, "Goto page") {
		return "1"
	}
	pageLinks := strings.Split(gensmallText, ",")
	return fmt.Sprintf("%d", len(pageLinks))
}
