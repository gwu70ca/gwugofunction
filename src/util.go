package main

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"net/http"
	"time"

	gofunc "github.com/gwu70ca/azuregofunctionhelper"
)

var greetingPage = `<html><body>Type your name here: <form action="/api/HttpExample" method="POST"><input type="text" name="name"><input type="submit" value="Submit"></form>
</body></html>`

var resultPage = `<html><body><h1>Welcome, {{.Name}}. You are running Azure Function app with Golang {{.GOVERSION}} on {{.GOOS}}<br><a href="Home">Home</a></h1>
</body></html>`

func _log(r *http.Request, m string) string {
	//Log can be found in: "/home/LogFiles/Application/Functions/Host"

	if m != "" {
		fmt.Println(m)
	}
	fmt.Println("==================================================")
	fmt.Printf("invocationid is: %s \n", r.Header.Get("X-Azure-Functions-InvocationId"))

	if r != nil {
		fmt.Printf("user agent is: %s \n", r.Header.Get("User-Agent"))
	}

	return time.Now().Format(time.RFC3339)
}

func queryParamsToString(queryParams *gofunc.DataHttpRequest) string {
	fmt.Println("queryParamsToString")
	var buffer bytes.Buffer

	for k, v := range queryParams.URL.Query() {
		fmt.Println("k:", k, "v:", v)
		buffer.WriteString(fmt.Sprintf("%v=%v,", k, v))
	}
	return buffer.String()
}

type Rss struct {
	XMLName xml.Name
	Version string      `xml:"version,attr"`
	Channel FeedChannel `xml:"channel"`
}

type FeedChannel struct {
	Title         string    `xml:"title"`
	Link          string    `xml:"link"`
	Description   string    `xml:"description"`
	Language      string    `xml:"language"`
	LastBuildDate string    `xml:"lastBuildDate"`
	Copyright     string    `xml:"copyright"`
	Items         []RssItem `xml:"item"`
}

type RssItem struct {
	Title       string `xml:"title"`
	Description string `xml:"description"`
	Link        string `xml:"link"`
	Category    string `xml:"category"`
	PubDate     string `xml:"pubDate"`
	Creator     string `xml:"creator"`
	Author      string `xml:"author"`
	DefaultDate *time.Time
}
