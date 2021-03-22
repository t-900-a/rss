package rss

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"time"
)

func parseAtom(data []byte) (*Feed, error) {
	warnings := false
	feed := AtomFeed{}
	p := xml.NewDecoder(bytes.NewReader(data))
	p.CharsetReader = charsetReader
	err := p.Decode(&feed)
	if err != nil {
		return nil, err
	}

	out := new(Feed)
	out.Title = feed.Title
	out.Description = feed.Description
	out.Author = feed.Author.Author()
	for _, link := range feed.Link {
		if link.Rel == "alternate" || link.Rel == "" {
			out.Link = link.Href
			break
		}
	}
	out.Image = feed.Image.Image()
	out.Refresh = time.Now().Add(10 * time.Minute)

	out.Items = make([]*Item, 0, len(feed.Items))
	out.ItemMap = make(map[string]struct{})

	// Process items.
	for _, item := range feed.Items {

		// Skip items already known.
		if _, ok := out.ItemMap[item.ID]; ok {
			continue
		}

		next := new(Item)
		next.Title = item.Title
		next.Summary = item.Summary
		next.Content = item.Content.RAWContent
		if item.Date != "" {
			next.Date, err = parseTime(item.Date)
			if err == nil {
				next.DateValid = true
			}
		}
		next.ID = item.ID
		for _, link := range item.Links {
			if link.Rel == "alternate" || link.Rel == "" {
				next.Link = link.Href
			} else {
				next.Enclosures = append(next.Enclosures, &Enclosure{
					URL:    link.Href,
					Type:   link.Type,
					Length: link.Length,
				})
			}
		}
		next.Read = false

		if next.ID == "" {
			if debug {
				fmt.Printf("[w] Item %q has no ID and will be ignored.\n", next.Title)
				fmt.Printf("[w] %#v\n", item)
			}
			warnings = true
			continue
		}

		if _, ok := out.ItemMap[next.ID]; ok {
			if debug {
				fmt.Printf("[w] Item %q has duplicate ID.\n", next.Title)
				fmt.Printf("[w] %#v\n", next)
			}
			warnings = true
			continue
		}

		out.Items = append(out.Items, next)
		out.ItemMap[next.ID] = struct{}{}
		out.Unread++
	}

	if warnings && debug {
		fmt.Printf("[i] Encountered warnings:\n%s\n", data)
	}

	return out, nil
}

type RAWContent struct {
	RAWContent string `xml:",innerxml"`
}

type AtomFeed struct {
	XMLName     xml.Name   `xml:"feed"`
	Title       string     `xml:"title"`
	Description string     `xml:"subtitle"`
	Author      AtomAuthor `xml:"author"`
	Link        []AtomLink `xml:"link"`
	Image       AtomImage  `xml:"image"`
	Items       []AtomItem `xml:"entry"`
	Updated     string     `xml:"updated"`
}

type AtomItem struct {
	XMLName   xml.Name   `xml:"entry"`
	Title     string     `xml:"title"`
	Summary   string     `xml:"summary"`
	Content   RAWContent `xml:"content"`
	Links     []AtomLink `xml:"link"`
	Date      string     `xml:"updated"`
	DateValid bool
	ID        string `xml:"id"`
}

type AtomImage struct {
	XMLName xml.Name `xml:"image"`
	Title   string   `xml:"title"`
	URL     string   `xml:"url"`
	Height  int      `xml:"height"`
	Width   int      `xml:"width"`
}

type AtomLink struct {
	Href   string `xml:"href,attr"`
	Rel    string `xml:"rel,attr"`
	Type   string `xml:"type,attr"`
	Length uint   `xml:"length,attr"`
}

type AtomAuthor struct {
	Name       string     `xml:"name"`
	URI        string     `xml:"uri"`
	Email      string     `xml:"email"`
	Extensions []AtomLink `xml:"link"`
}

func (a *AtomImage) Image() *Image {
	out := new(Image)
	out.Title = a.Title
	out.URL = a.URL
	out.Height = uint32(a.Height)
	out.Width = uint32(a.Width)
	return out
}

func (a *AtomAuthor) Author() *Author {
	e := make([]*Link, 0, len(a.Extensions))
	l := new(Link)
	for _, ext := range a.Extensions {
		l = nil
		l = new(Link)
		l.Href = ext.Href
		l.Rel = ext.Rel
		l.Type = ext.Type
		e = append(e, l)
	}
	out := new(Author)
	out.Name = a.Name
	out.URI = a.URI
	out.Email = a.Email
	out.Extensions = e
	return out
}
