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
	"regexp"
	"time"
)

var (
	token   string
	mention = regexp.MustCompile(`<@\S+>\s*(\S*)`)
)

func init() {
	token = os.Getenv("OAUTH_ACCESS_TOKEN")
	if token == "" {
		log.Fatalf("OAUTH_ACCESS_TOKEN is empty")
	}
}

type Challenge struct {
	Token     string `json:"token"`
	Challenge string `json:"challenge"`
	Type      string `json:"type"`
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
	res, err := googleTrendsBot(w, r)
	if err != nil {
		w.Write([]byte(err.Error()))
		w.WriteHeader(400)
		return
	}
	if res != "" {
		w.Write([]byte(res))
	}
	w.WriteHeader(200)
}

func googleTrendsBot(w http.ResponseWriter, r *http.Request) (string, error) {
	d, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return "", err
	}

	fmt.Println("Request Body:", string(d))

	c := &Challenge{}
	err = json.Unmarshal(d, c)
	fmt.Println("Challenge Error:", err)
	if c.Challenge != "" {
		return c.Challenge, nil
	}

	body := &Body{}
	err = json.Unmarshal(d, body)
	fmt.Println("Body Error:", err)
	if err != nil {
		return "", err
	}

	return "", FetchAndPostTrends(body.Event)
}

func FetchAndPostTrends(e Event) error {
	fmt.Printf("FetchAndPostTrends: %q\n", e)

	matched := mention.FindStringSubmatch(e.Text)
	if matched[1] == "" {
		return fmt.Errorf("geo not found")
	}

	ts, err := Fetch(matched[1])
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
	u := fmt.Sprintf("https://trends.google.co.jp/trends/trendingsearches/daily/rss?geo=%s", geo)

	fmt.Printf("Fetch: %s\n", u)

	resp, err := http.Get(u)
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

	// trends := []Trend{}
	// today := time.Now()
	// for _, trend := range rss.Channel.Items {
	// 	t, err := time.Parse("Mon, 2 Jan 2006 15:04:05 -0700", trend.PubDate)
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// 	fmt.Println(t, today)
	// 	if t.Year() == today.Year() && t.Month() == today.Month() && t.Day() == today.Day() {
	// 		trends = append(trends, trend)
	// 	}
	// }

	return rss.Channel.Items, nil
}

func PostMessage(trends []Trend, channel string) error {
	fmt.Printf("PostMessage: %q, %s\n", trends, channel)

	as := []Attachment{}
	for i, t := range trends {
		date, err := time.Parse("Mon, 2 Jan 2006 15:04:05 -0700", t.PubDate)
		if err != nil {
			return err
		}

		a := Attachment{
			Title:    fmt.Sprintf("%d. %s", i+1, t.Title),
			ThumbURL: t.Picture,
			Fields: []Field{
				Field{
					Title: "Date",
					Value: date.Format("1/2"),
					Short: true,
				},
				Field{
					Title: "Approx Traffic",
					Value: t.ApproxTraffic,
					Short: true,
				},
			},
		}
		if len(t.NewsItems) > 0 {
			a.TitleLink = t.NewsItems[0].URL
		}
		if t.Description != "" {
			a.Text = t.Description
		}

		as = append(as, a)
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
	Fallback   string  `json:"fallback"`    //"Required plain-text summary of the attachment."
	Color      string  `json:"color"`       //"#2eb886"
	Pretext    string  `json:"pretext"`     //"Optional text that appears above the attachment block"
	AuthorName string  `json:"author_name"` //"Bobby Tables"
	AuthorLink string  `json:"author_link"` //"http://flickr.com/bobby/"
	AuthorIcon string  `json:"author_icon"` //"http://flickr.com/icons/bobby.jpg"
	Title      string  `json:"title"`       //"Slack API Documentation"
	TitleLink  string  `json:"title_link"`  //"https://api.slack.com/"
	Text       string  `json:"text"`        //"Optional text that appears within the attachment"
	Fields     []Field `json:"fields"`
	ImageURL   string  `json:"image_url"`   //"http://my-website.com/path/to/image.jpg"
	ThumbURL   string  `json:"thumb_url"`   //"http://example.com/path/to/thumb.png"
	Footer     string  `json:"footer"`      //"Slack API"
	FooterIcon string  `json:"footer_icon"` //"https://platform.slack-edge.com/img/default_application_icon.png"
	TS         string  `json:"ts"`          //12345678
}

type Field struct {
	Title string `json:"title"` //"Priority"
	Value string `json:"value"` //"High"
	Short bool   `json:"short"` //fals
}
