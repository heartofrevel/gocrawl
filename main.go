package main

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"sync"

	"github.com/labstack/echo"
	"golang.org/x/net/html"
)

var extensionList = []string{".pdf", ".doc", ".docx", ".ppt", ".jpeg", ".jpg", ".png"}

func getReference(token html.Token) (flag bool, url string) {
	for _, a := range token.Attr {
		if a.Key == "href" {
			url = a.Val
			flag = true
		}
	}
	return
}

func checkExt(ext string) bool {
	for _, item := range extensionList {
		if item == ext {
			return true
		}
	}
	return false
}

func urlInURLList(url string, urlList *[]string) bool {
	for _, item := range *urlList {
		if item == url {
			return true
		}
	}
	return false
}

func crawler(c echo.Context, urlRec string, feed chan string, urlList *[]string, wg *sync.WaitGroup) {
	defer wg.Done()
	URL, _ := url.Parse(urlRec)
	response, err := http.Get(urlRec)
	if err != nil {
		log.Print(err)
		return
	}

	body := response.Body
	defer body.Close()

	tokenizer := html.NewTokenizer(body)
	flag := true
	for flag {
		tokenType := tokenizer.Next()
		switch {
		case tokenType == html.ErrorToken:
			flag = false
			break
		case tokenType == html.StartTagToken:
			token := tokenizer.Token()

			// Check if the token is an <a> tag
			isAnchor := token.Data == "a"
			if !isAnchor {
				continue
			}

			ok, urlHref := getReference(token)
			if !ok {
				continue
			}

			// Make sure the url begines in http**
			hasProto := strings.Index(urlHref, "http") == 0
			if hasProto {
				if !urlInURLList(urlHref, urlList) {
					if strings.Contains(urlHref, URL.Host) {
						*urlList = append(*urlList, urlHref)
						fmt.Println(urlHref)
						c.String(http.StatusOK, urlHref+"\n")
						if !checkExt(filepath.Ext(urlHref)) {
							wg.Add(1)
							go crawler(c, urlHref, feed, urlList, wg)
						}
					}
				}
			}
		}
	}
}

func scrape(c echo.Context) error {
	var urlList []string
	var wg sync.WaitGroup
	urlParam := c.FormValue("url")
	feed := make(chan string, 1000)
	wg.Add(1)
	go crawler(c, urlParam, feed, &urlList, &wg)
	wg.Wait()
	return c.String(http.StatusOK, "Complete")
}

func main() {
	fmt.Println("Starting Web Scraper API")
	e := echo.New()
	e.POST("/scraper", scrape)
	e.Start(":8001")
}
