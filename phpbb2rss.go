package phpbb2rss

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"net/http"
	"net/url"
	"strings"
	"text/template"
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
	Author      string `xml:"author"`
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

	descTemplate := `Category: {{.Category}}
Author: {{.Author}}
Last Commenter: {{.LastCommenter}}
Replies: {{.Replies}}
Posts: {{.Posts}}
Pages: {{.Pages}}`

	tmpl, err := template.New("description").Parse(descTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse description template: %w", err)
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

		topicURL, err := baseURL.Parse(topicLink)
		if err != nil {
			return
		}

		link := topicURL.String()
		q := topicURL.Query()
		q.Del("sid")
		topicURL.RawQuery = q.Encode()
		guid := topicURL.String()

		if latestPostExists {
			latestPostURL, err := baseURL.Parse(latestPostLink)
			if err != nil {
				return
			}
			link = latestPostURL.String()
			q := latestPostURL.Query()
			q.Del("sid")
			latestPostURL.RawQuery = q.Encode()
			guid = latestPostURL.String()
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
		category := strings.TrimSpace(s.Find("a.forumlink").Text())

		var descriptionBuilder bytes.Buffer
		if err := tmpl.Execute(&descriptionBuilder, map[string]string{
			"Category":      category,
			"Author":        author,
			"LastCommenter": lastCommenter,
			"Replies":       replies,
			"Posts":         posts,
			"Pages":         pages,
		}); err != nil {
			return
		}

		titleWithCategory := fmt.Sprintf("[%s] %s", category, title)

		items = append(items, Item{
			Title:       titleWithCategory,
			Link:        link,
			Description: descriptionBuilder.String(),
			PubDate:     parsedDate.Format(time.RFC1123),
			GUID:        guid,
			Author:      author,
		})
	})

	rss := RSS{
		Version: "2.0",
		Channel: Channel{
			Title:       pageTitle,
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
