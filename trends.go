package trends

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"
)

var (
	token string
)

func init() {
	token = os.Getenv("OAUTH_ACCESS_TOKEN")
	if token == "" {
		log.Fatalf("OAUTH_ACCESS_TOKEN is empty")
	}
}

type Body struct {
	Event Event `json:"event"`
}

type Event struct {
	ClientMsgID string `json:"client_msg_id"`
	Text        string `json:"text"`
	Channel     string `json:"channel"`
}

func GoogleTrendsBot(w http.ResponseWriter, r *http.Request) {
	err := googleTrendsBot(w, r)
	if err != nil {
		w.Write([]byte(err.Error()))
		w.WriteHeader(400)
		return
	}
	w.WriteHeader(200)
}

func googleTrendsBot(w http.ResponseWriter, r *http.Request) error {
	d, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return err
	}
	body := &Body{}
	err = json.Unmarshal(d, body)
	if err != nil {
		return err
	}

	fmt.Printf("Request Body: %+v\n", body)

	return FetchAndPostTrends(body.Event)
}

func FetchAndPostTrends(e Event) error {
	ts, err := Fetch(e.Text)
	if err != nil {
		return err
	}
	return PostMessage(ts, e.Channel)
}

type RSS struct {
	Channel Channel `xml:"channel"`
}

type Channel struct {
	Title       string // Daily Search Trends
	Description string `xml:"description"` // Recent searches
	Link        string `xml:"link"`        // https://trends.google.co.jp/trends/trendingsearches/daily?geo=JP
	// AtomLink `xml:"atom:link,attr"` // https://trends.google.co.jp/trends/trendingsearches/daily/rss?geo=JP
	Items []Trend `xml:"item"`
}

type Trend struct {
	Title         string     `xml:"title"`                                                                    // Copa America
	ApproxTraffic string     `xml:"https://trends.google.co.jp/trends/trendingsearches/daily approx_traffic"` // 500,000+
	Description   string     `xml:"description"`                                                              // fox sports, Venezuela vs Argentina, argentina copa america, argentina vs venezuela 2019
	Link          string     `xml:"link"`                                                                     // https://trends.google.co.jp/trends/trendingsearches/daily?geo=US#Copa%20America
	PubDate       string     `xml:"pubDate"`                                                                  // Fri, 28 Jun 2019 13:00:00 -0700
	Picture       string     `xml:"https://trends.google.co.jp/trends/trendingsearches/daily picture"`        // https://t2.gstatic.com/images?q=tbn:ANd9GcQHkjNN41ODsg8kdwlXugG21c9CeDnfVu5YMbT8PgPxYPOvFynG8kNlLzCMDGQ53U1z6nljwIXG
	PictureSource string     `xml:"https://trends.google.co.jp/trends/trendingsearches/daily picture_source"` // BBC Sport
	NewsItems     []NewsItem `xml:"https://trends.google.co.jp/trends/trendingsearches/daily news_item"`
}

type NewsItem struct {
	Title   string `xml:"https://trends.google.co.jp/trends/trendingsearches/daily news_item_title"`   // <b>Copa America</b> quarter-finals - Argentina set up Brazil semi-final
	Snippet string `xml:"https://trends.google.co.jp/trends/trendingsearches/daily news_item_snippet"` // Lionel Messi was quiet for much of the game but he won&#39;t mind as Argentina made it through. Have a read of the report. We will return for more <b>Copa America</b> action during the semi-finals. Until then. Article Reactions. Like. 4 likes4. Dislike. 0 dislikes0.
	URL     string `xml:"https://trends.google.co.jp/trends/trendingsearches/daily news_item_url"`     // https://www.bbc.co.uk/sport/live/football/48766803
	Source  string `xml:"https://trends.google.co.jp/trends/trendingsearches/daily news_item_source"`  // BBC Sport
}

func Fetch(geo string) ([]Trend, error) {
	resp, err := http.Get(fmt.Sprintf("https://trends.google.co.jp/trends/trendingsearches/daily/rss?geo=%s", geo))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	rss := RSS{}
	err = xml.Unmarshal(body, &rss)
	if err != nil {
		return nil, err
	}

	trends := []Trend{}
	today := time.Now()
	for _, trend := range rss.Channel.Items {
		t, err := time.Parse("Mon, 2 Jan 2006 15:04:05 -0700", trend.PubDate)
		if err != nil {
			return nil, err
		}
		if t.Year() == today.Year() && t.Month() == today.Month() && t.Day() == today.Day() {
			trends = append(trends, trend)
		}
	}

	return trends, nil
}

func PostMessage(trends []Trend, channel string) error {
	as := []Attachment{}
	for i, t := range trends {
		as = append(as, Attachment{
			Title:    fmt.Sprintf("%d. %s (%s)", i+1, t.Title, t.ApproxTraffic),
			Text:     t.Description,
			ImageURL: t.Picture,
		})
	}
	// text := strings.Join(ts, "\n")

	m := Message{
		Channel:     channel,
		Attachments: as,
	}
	b, err := json.Marshal(m)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", "https://slack.com/api/chat.postMessage", bytes.NewBuffer(b))
	if err != nil {
		return err
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Add("Content-Type", "application/json; charset=utf-8")
	cli := &http.Client{}
	res, err := cli.Do(req)
	if err != nil {
		return err
	}
	fmt.Println(res)
	return nil
}

type Message struct {
	Channel     string       `json:"channel"`
	Text        string       `json:"text"`
	Markdown    bool         `json:"mrkdwn"`
	Attachments []Attachment `json:"attachments"`
}

type Attachment struct {
	Title    string `json:"title"`
	Text     string `json:"text"`
	ImageURL string `json:"image_url"`
}
