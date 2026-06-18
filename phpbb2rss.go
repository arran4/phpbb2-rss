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
	if isPHPBB3(doc) {
		items = parsePHPBB3(doc, baseURL, tmpl)
	} else if isPHPBB2(doc) {
		items = parsePHPBB2(doc, baseURL, tmpl)
	}

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

func isPHPBB3(doc *goquery.Document) bool {
	return doc.Find("ul.topiclist.topics").Length() > 0
}

func isPHPBB2(doc *goquery.Document) bool {
	return doc.Find(".forumline").Length() > 0
}

func parsePHPBB2(doc *goquery.Document, baseURL *url.URL, tmpl *template.Template) []Item {
	var items []Item
	doc.Find(".forumline tr").Each(func(i int, s *goquery.Selection) {
		topic := s.Find(".topictitle a").First()
		title := strings.TrimSpace(topic.Text())
		topicLink, topicExists := topic.Attr("href")
		latestPostLink, latestPostExists := s.Find("a:contains('View latest post')").Attr("href")
		if !latestPostExists {
			latestPostLink, latestPostExists = s.Find("a:has(img[alt='View latest post'])").Attr("href")
		}
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

		pubDateRaw := strings.TrimSpace(s.Find(".postdetails").Last().Contents().First().Text())
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
	return items
}

func parsePHPBB3(doc *goquery.Document, baseURL *url.URL, tmpl *template.Template) []Item {
	var items []Item
	doc.Find("ul.topiclist.topics li.row").Each(func(i int, s *goquery.Selection) {
		topic := s.Find("a.topictitle").First()
		title := strings.TrimSpace(topic.Text())
		topicLink, topicExists := topic.Attr("href")
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

		latestPostLink, latestPostExists := s.Find("dd.lastpost a[title='Go to last post']").Attr("href")
		if !latestPostExists {
			latestPostLink, latestPostExists = s.Find("dd.lastpost span a").Last().Attr("href")
		}

		if latestPostExists {
			latestPostURL, err := baseURL.Parse(latestPostLink)
			if err == nil {
				link = latestPostURL.String()
				q := latestPostURL.Query()
				q.Del("sid")
				latestPostURL.RawQuery = q.Encode()
				guid = latestPostURL.String()
			}
		}

		pubDateRaw, hasDatetime := s.Find("dd.lastpost time").Attr("datetime")
		var parsedDate time.Time
		if hasDatetime {
			parsedDate, err = time.Parse(time.RFC3339, pubDateRaw)
			if err != nil {
				parsedDate = time.Time{}
			}
		} else {
			parsedDate = time.Time{}
		}

		author := strings.TrimSpace(s.Find(".list-inner .responsive-hide a.username, .list-inner .responsive-hide a.username-coloured").First().Text())
		if author == "" {
			author = strings.TrimSpace(s.Find(".list-inner .responsive-hide a").First().Text())
		}

		lastCommenter := strings.TrimSpace(s.Find("dd.lastpost a.username, dd.lastpost a.username-coloured").First().Text())
		if lastCommenter == "" {
			lastCommenter = strings.TrimSpace(s.Find("dd.lastpost > span > a").First().Text())
		}

		category := strings.TrimSpace(s.Find(".list-inner .responsive-hide a").Last().Text())
		if category == "" || category == author {
			category = strings.TrimSpace(s.Find(".responsive-show a[href*='viewforum.php']").First().Text())
		}

		repliesNode := s.Find("dd.posts")
		repliesNodeClone := repliesNode.Clone()
		repliesNodeClone.Find("dfn").Remove()
		replies := strings.TrimSpace(repliesNodeClone.Text())

		viewsNode := s.Find("dd.views")
		viewsNodeClone := viewsNode.Clone()
		viewsNodeClone.Find("dfn").Remove()
		posts := strings.TrimSpace(viewsNodeClone.Text())

		pages := ""
		pagination := s.Find(".pagination")
		if pagination.Length() > 0 {
			lastPage := pagination.Find("ul li a.button").Last().Text()
			if lastPage != "" {
				pages = lastPage
			} else {
				pages = "1"
			}
		} else {
			pages = "1"
		}

		var descriptionBuilder bytes.Buffer
		if err := tmpl.Execute(&descriptionBuilder, map[string]string{
			"Category":      category,
			"Author":        author,
			"LastCommenter": lastCommenter,
			"Replies":       replies,
			"Posts":         posts, // Using views for posts here as it aligns with old format "views" display if any
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
	return items
}
