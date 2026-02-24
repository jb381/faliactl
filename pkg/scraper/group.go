package scraper

import (
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// FetchGroups retrieves all the available groups from the main schedule.html page
func (c *Client) FetchGroups() ([]Group, error) {
	resp, err := c.Get("schedule.html")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}

	var groups []Group

	// The groups are stored as <option> tags inside a <select id="group">
	doc.Find("select#group option").Each(func(i int, sel *goquery.Selection) {
		val, exists := sel.Attr("value")
		if exists && val != "" {
			name := strings.TrimSpace(sel.Text())
			groups = append(groups, Group{
				Name: name,
				URL:  val,
			})
		}
	})

	return groups, nil
}
