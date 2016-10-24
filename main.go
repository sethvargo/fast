package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/pkg/errors"

	"golang.org/x/net/html"
)

const (
	HTML_URL string = "https://fast.com"
	API_URL  string = "https://api.fast.com/netflix/speedtest"
)

var spaces = strings.Repeat(" ", 50)

// tokenRegexp is the pattern to search for in the js to get the API token for
// making the request.
var tokenRegexp = regexp.MustCompile(`token:"([a-zA-Z0-9]+)"`)

func main() {
	os.Exit(realMain())
}

func realMain() int {
	fmt.Println("Getting API token...")

	s, err := appJSLink(HTML_URL)
	if err != nil {
		err = errors.Wrap(err, "getting app JS link failed")
		log.Fatal(err)
	}

	s, err = appJS(s)
	if err != nil {
		err = errors.Wrap(err, "getting app JS")
		log.Fatal(err)
	}

	token, err := extractToken(s)
	if err != nil {
		err = errors.Wrap(err, "extracting token failed")
		log.Fatal(err)
	}

	list, err := fastURLs(API_URL + "?https=true&token=" + token)
	if err != nil {
		err = errors.Wrap(err, "getting urls failed")
		log.Fatal(err)
	}

	u, err := url.Parse(list[0])
	if err != nil {
		err = errors.Wrap(err, "parsing url failed")
		log.Fatal(err)
	}

	u.Path = "/speedtest/range/0-2048"
	first := u.String()

	u.Path = "/speedtest/range/0-26214400"
	main := u.String()

	d, err := NewDownloader(&NewDownloaderInput{
		URLs: []string{first, main},
	})
	if err != nil {
		err = errors.Wrap(err, "creating downloader failed")
		log.Fatal(err)
	}

	fmt.Println("Starting download test...")
	fmt.Println()

	spin := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	spin.Prefix = "\t"
	spin.Suffix = "  Connecting to server...\r"
	spin.Start()

	go func() {
		for metric := range d.MetricsCh() {
			spin.Suffix = fmt.Sprintf("  %s%s\r", metric, spaces)
		}
	}()

	metric, err := d.DownloadAll()
	if err != nil {
		err = errors.Wrap(err, "downloading failed")
		log.Fatal(err)
	}

	spin.Stop()

	fmt.Printf("\rSpeed: %s%s\n", metric, spaces)

	return 0
}

func appJSLink(u string) (string, error) {
	resp, err := http.Get(u)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	z := html.NewTokenizer(resp.Body)
	for {
		tt := z.Next()

		switch tt {
		case html.ErrorToken:
			return "", z.Err()
		case html.StartTagToken:
			tn, hasAttr := z.TagName()
			if string(tn) == "script" && hasAttr {
				for {
					attr, val, more := z.TagAttr()
					if string(attr) == "src" {
						return u + string(val), nil
					}
					if !more {
						break
					}
				}
			}
		default:
			continue
		}
	}
}

func appJS(u string) (string, error) {
	resp, err := http.Get(u)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var b bytes.Buffer
	if _, err := io.Copy(&b, resp.Body); err != nil {
		return "", err
	}
	return b.String(), nil
}

// extractToken uses the tokenRegexp to extract the token from the given
// payload.
func extractToken(s string) (string, error) {
	matches := tokenRegexp.FindAllStringSubmatch(s, 1)
	if len(matches) == 0 || len(matches[0]) == 0 {
		return "", fmt.Errorf("Could not find token in string!")
	}
	return matches[0][1], nil
}

// fastURLs extracts the list of download URLs from the given url.
func fastURLs(u string) ([]string, error) {
	resp, err := http.Get(u)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	dest := make([]map[string]string, 0, 5)
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&dest); err != nil {
		return nil, err
	}

	list := make([]string, len(dest))
	for i, u := range dest {
		list[i] = u["url"]
	}
	return list, nil
}

// mkfifo /tmp/fast.com.test.fifo ;token=$(curl -s https://fast.com/app-ed402d.js|egrep -om1 'token:"[^"]+'|cut -f2 -d'"'); curl -s "https://api.fast.com/netflix/speedtest?https=true&token=$token" |egrep -o 'https[^"]+'|while read url; do first=${url/speedtest/speedtest\/range\/0-2048}; next=${url/speedtest/speedtest\/range\/0-26214400};(curl -s -H 'Referer: https://fast.com/' -H 'Origin: https://fast.com' "$first" > /tmp/fast.com.test.fifo; for i in {1..10}; do curl -s -H 'Referer: https://fast.com/' -H 'Origin: https://fast.com'  "$next">>/tmp/fast.com.test.fifo; done)& done & pv /tmp/fast.com.test.fifo > /dev/null; rm /tmp/fast.com.test.fifo
