package phpbb2rss

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestParsePHPBB2(t *testing.T) {
	html := `
	<html>
	<head><title>Old Forum</title></head>
	<body>
		<a class="nav" href="#">Forum Index</a>
		<table class="forumline">
			<tr>
				<td class="row1"><span class="topictitle"><a href="viewtopic.php?t=123&sid=123">Old Topic 1</a></span></td>
				<td class="row2"><span class="postdetails">10</span></td>
				<td class="row3"><span class="postdetails">100</span></td>
				<td class="row1"><span class="name"><a href="#">OldAuthor</a></span></td>
				<td class="row2"><span class="postdetails">Mon Jan 02, 2006 3:04 pm<br /><a href="profile.php">OldLastCommenter</a> <a href="viewtopic.php?t=123&p=456#456"><img src="icon_latest_reply.gif" alt="View latest post" /></a></span></td>
				<td><span class="gensmall"></span></td>
				<td><a class="forumlink" href="#">Old Category</a></td>
			</tr>
		</table>
	</body>
	</html>
	`

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(html))
	}))
	defer ts.Close()

	rss, err := FetchAndGenerateRSS(ts.URL)
	if err != nil {
		t.Fatalf("Failed to generate RSS: %v", err)
	}

	if !strings.Contains(rss, "Old Topic 1") {
		t.Errorf("Expected RSS to contain 'Old Topic 1', got: %s", rss)
	}
	if !strings.Contains(rss, "OldAuthor") {
		t.Errorf("Expected RSS to contain 'OldAuthor'")
	}
}

func TestParsePHPBB3(t *testing.T) {
	html := `
	<html>
	<head><title>New Forum</title></head>
	<body>
		<a class="nav" href="#">Forum Index</a>
		<ul class="topiclist topics">
			<li class="row bg1">
				<div class="list-inner">
					<a href="viewtopic.php?t=456&sid=123" class="topictitle">New Topic 1</a>
					<div class="responsive-hide left-box">
						by <a href="#" class="username">NewAuthor</a> &raquo;
						in <a href="viewforum.php">New Category</a>
					</div>
				</div>
				<dd class="posts">15 <dfn>Replies</dfn></dd>
				<dd class="views">250 <dfn>Views</dfn></dd>
				<dd class="lastpost">
					<span>
						<dfn>Last post </dfn>by <a href="#" class="username-coloured">NewLastCommenter</a>
						<a href="viewtopic.php?p=789#p789" title="Go to last post">Icon</a>
						<br /><time datetime="2026-04-14T10:21:19+00:00">Tue Apr 14, 2026 10:21 am</time>
					</span>
				</dd>
			</li>
		</ul>
	</body>
	</html>
	`

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(html))
	}))
	defer ts.Close()

	rss, err := FetchAndGenerateRSS(ts.URL)
	if err != nil {
		t.Fatalf("Failed to generate RSS: %v", err)
	}

	if !strings.Contains(rss, "New Topic 1") {
		t.Errorf("Expected RSS to contain 'New Topic 1', got: %s", rss)
	}
	if !strings.Contains(rss, "NewAuthor") {
		t.Errorf("Expected RSS to contain 'NewAuthor'")
	}
}
