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

type urlFound struct {
	Count     int      `json:"count"`
	Links     []string `json:"links"`
	Images    []string `json:"images"`
	Documents []string `json:"docs"`
}

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
						// c.String(http.StatusOK, urlHref+"\n")Documents
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

func scrapePOST(c echo.Context) error {
	var urlList []string
	urlSession := urlFound{}
	var wg sync.WaitGroup
	urlParam := c.FormValue("url")
	feed := make(chan string, 1000)
	wg.Add(1)
	go crawler(c, urlParam, feed, &urlList, &wg)
	wg.Wait()
	var count = 0
	for _, url := range urlList {
		if filepath.Ext(url) == ".jpg" || filepath.Ext(url) == ".jpeg" || filepath.Ext(url) == ".png" {
			urlSession.Images = append(urlSession.Images, url)
		} else if filepath.Ext(url) == ".doc" || filepath.Ext(url) == ".docx" || filepath.Ext(url) == ".pdf" || filepath.Ext(url) == ".ppt" {
			urlSession.Documents = append(urlSession.Documents, url)
		} else {
			urlSession.Links = append(urlSession.Links, url)
		}
		count = count + 1
	}
	urlSession.Count = count
	// jsonResp, _ := json.Marshal(urlSession)
	// fmt.Print(urlSession)
	return c.JSON(http.StatusOK, urlSession)
}

func scrapeGET(c echo.Context) error {
	var urlList []string
	urlSession := &urlFound{}
	var wg sync.WaitGroup
	urlParam := c.QueryParam("url")
	feed := make(chan string, 1000)
	wg.Add(1)
	go crawler(c, urlParam, feed, &urlList, &wg)
	wg.Wait()
	var count = 0
	for _, url := range urlList {
		if filepath.Ext(url) == ".jpg" || filepath.Ext(url) == ".jpeg" || filepath.Ext(url) == ".png" {
			urlSession.Images = append(urlSession.Images, url)
		} else if filepath.Ext(url) == ".doc" || filepath.Ext(url) == ".docx" || filepath.Ext(url) == ".pdf" || filepath.Ext(url) == ".ppt" {
			urlSession.Documents = append(urlSession.Documents, url)
		} else {
			urlSession.Links = append(urlSession.Links, url)
		}
		count = count + 1
	}
	urlSession.Count = count
	// jsonResp, _ := json.Marshal(urlSession)
	return c.JSON(http.StatusOK, urlSession)
}

func main() {
	fmt.Println("Starting Web Scraper API")
	e := echo.New()
	e.POST("/scraper", scrapePOST)
	e.GET("/scraper", scrapeGET)
	e.Start(":8001")
}
