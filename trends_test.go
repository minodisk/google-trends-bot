package trends_test

import (
	"fmt"
	"testing"

	trends "github.com/minodisk/google-trends-bot"
)

func TestFetchAndPostTrends(t *testing.T) {
	ts, err := trends.Fetch("JP")
	if err != nil {
		t.Fatal(err)
	}
	err = trends.PostMessage(ts)
	if err != nil {
		t.Fatal(err)
	}
}

func TestFetch(t *testing.T) {
	for _, geo := range []string{
		"US",
		"JP",
	} {
		fmt.Println("=============")
		fmt.Println(geo)
		ts, err := trends.Fetch(geo)
		if err != nil {
			t.Fatal(err)
		}
		for _, trend := range ts {
			fmt.Println(trend.NewsItems)
		}
	}
}

func TestPostMessage(t *testing.T) {
	err := trends.PostMessage(nil)
	if err != nil {
		t.Fatal(err)
	}
}
