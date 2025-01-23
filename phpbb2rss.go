package phpbb2rss

import (
	"encoding/xml"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"net/http"
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

	var items []Item
	doc.Find(".forumline tr").Each(func(i int, s *goquery.Selection) {
		topic := s.Find(".topictitle a").First()
		title := strings.TrimSpace(topic.Text())
		topicLink, topicExists := topic.Attr("href")
		latestPostLink, latestPostExists := s.Find("a:contains('View latest post')").Attr("href")
		if !topicExists || title == "" {
			return
		}

		link := fmt.Sprintf("%s/%s", forumURL, topicLink)
		if latestPostExists {
			link = fmt.Sprintf("%s/%s", forumURL, latestPostLink)
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
			Title:       "PHPBB2 Forum Topics",
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
