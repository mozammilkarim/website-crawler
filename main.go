package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

//to make the websites's web server believe that this is not a bot and a genuine user
// important for web scraping
var userAgents = []string{
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/109.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/109.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/108.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/108.0.0.0 Safari/537.36",
	"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/108.0.0.0 Safari/537.36",
}

func randomUserAgent() string {
	rand.Seed(time.Now().Unix())
	randNum := rand.Int() % len(userAgents)
	return userAgents[randNum]
}
func getRequest(targetUrl string) (*http.Response, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", targetUrl, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", randomUserAgent())
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	} else {
		return res, nil
	}
}
func discoverLinks(response *http.Response, baseUrl string) []string {
	if response != nil {
		doc, _ := goquery.NewDocumentFromResponse(response)
		foundUrls := []string{}
		if doc != nil {
			doc.Find("a").Each(func(i int, s *goquery.Selection) {
				res, _ := s.Attr("href")
				foundUrls = append(foundUrls, res)
			})
		}
		return foundUrls
	}
	return []string{}
}

var tokens = make(chan struct{}, 5) // at any point, max 5 threads can run
func checkRelative(href string, baseUrl string) string {
	if strings.HasPrefix(href, "/") { //if relative path is there attach baseUrl, otherwise nothing
		return fmt.Sprintf("%s%s", baseUrl, href)
	} else {
		return href
	}

}
func resolveRelativeLinks(href string, baseUrl string) (bool, string) {
	resultHref := checkRelative(href, baseUrl)
	baseParse, err := url.Parse(baseUrl)
	if err != nil {
		fmt.Println("Error ocurred", err)
	}
	resultParse, err := url.Parse(resultHref)
	if err != nil {
		fmt.Println("Error ocurred", err)
	}
	if baseParse != nil && resultParse != nil {
		if baseParse.Host == resultParse.Host {
			return true, resultHref
		} else {
			return false, ""
		}
	}
	return false, ""
}
func crawl(targetUrl string, baseUrl string) []string {
	fmt.Println(targetUrl)
	tokens <- struct{}{} // i don't know this
	resp, _ := getRequest(targetUrl)
	<-tokens
	links := discoverLinks(resp, baseUrl)

	foundUrls := []string{}

	for _, link := range links {
		ok, correctLink := resolveRelativeLinks(link, baseUrl)
		if ok {
			if correctLink != "" {
				foundUrls = append(foundUrls, correctLink)
			}
		}
	}
	return foundUrls
}
func main() {
	baseUrl := "https://theguardian.com"
	worklist := make(chan []string) //creates a channel for routines, so that routines can
	//interact concurrently
	var n = 1
	//creates a go routine, a parallel thread to run, initializing the channel with baseUrl
	go func() { worklist <- []string{baseUrl} }()
	//to keep a track of links visited
	seen := make(map[string]bool)

	for ; n > 0; n-- {
		//pick items from worklist as it will be changed continously,channel
		list := <-worklist

		for _, link := range list {
			if !seen[link] {
				seen[link] = true
				//n represents total no. of links visited
				n++
				go func(link string, baseUrl string) {
					foundLinks := crawl(link, baseUrl)
					//if another link is found after crawl , add to channel
					if foundLinks != nil {
						worklist <- foundLinks
					}
				}(link, baseUrl)
			}
		}
	}

}
