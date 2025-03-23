package main

import (
	"bufio"
	"encoding/xml"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type RSS struct {
	Channel struct {
		Items []struct {
			Title     string `xml:"title"`
			PubDate   string `xml:"pubDate"`
			Enclosure struct {
				URL string `xml:"url,attr"`
			} `xml:"enclosure"`
		} `xml:"item"`
	} `xml:"channel"`
}

type writeCounter struct {
	progress chan<- int64
}

func (wc *writeCounter) Write(p []byte) (int, error) {
	n := len(p)
	wc.progress <- int64(n)
	return n, nil
}

func main() {
	// Define the --all flag
	downloadAll := flag.Bool("all", false, "Download all episodes without prompting")
	flag.Parse()

	if len(flag.Args()) < 1 {
		fmt.Println("Usage: download_rss_episodes [--all] <rss_feed_url>")
		os.Exit(1)
	}

	rssFeedURL := flag.Args()[0]

	resp, err := http.Get(rssFeedURL)
	if err != nil {
		fmt.Println("Error fetching RSS feed:", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading RSS feed:", err)
		os.Exit(1)
	}

	var rss RSS
	if err := xml.Unmarshal(body, &rss); err != nil {
		fmt.Println("Error parsing RSS feed:", err)
		os.Exit(1)
	}

	for _, item := range rss.Channel.Items {
		pubDate, err := time.Parse(time.RFC1123Z, item.PubDate)
		if err != nil {
			fmt.Println("Error parsing publication date:", err)
			continue
		}

		cleanedTitle := strings.Map(func(r rune) rune {
			if strings.ContainsRune("/\\?%*:|\"<>", r) {
				return '-'
			}
			return r
		}, item.Title)
		if strings.HasSuffix(cleanedTitle, "-") {
			cleanedTitle = strings.TrimSuffix(cleanedTitle, "-")
		}
		filename := fmt.Sprintf("%s-%s.mp3", pubDate.Format("20060102"), cleanedTitle)

		if _, err := os.Stat(filename); err == nil {
			fmt.Printf("File '%s' already exists. Skipping download.\n", filename)
			continue
		}

		// Skip prompt if --all is set
		if !*downloadAll {
			reader := bufio.NewReader(os.Stdin)
			fmt.Printf("Do you want to download '%s'? (y/n/q) ", filename)
			answer, _ := reader.ReadString('\n')
			answer = strings.TrimSpace(answer)

			if answer == "n" {
				fmt.Printf("Skipped '%s'\n", item.Title)
				continue
			} else if answer == "q" {
				break
			}
		} else {
			fmt.Printf("Downloading '%s'\n", filename)
		}

		out, err := os.Create(filename)
		if err != nil {
			fmt.Println("Error creating file:", err)
			continue
		}
		defer out.Close()

		resp, err := http.Get(item.Enclosure.URL)
		if err != nil {
			fmt.Println("Error downloading episode:", err)
			continue
		}
		defer resp.Body.Close()
		totalSize := resp.ContentLength
		progress := make(chan int64)

		go func() {
			var downloadedSize int64
			for p := range progress {
				downloadedSize += p
				fmt.Printf("\rDownloading... %.1f%% complete", float64(downloadedSize)/float64(totalSize)*100)
			}
		}()

		_, err = io.Copy(out, io.TeeReader(resp.Body, &writeCounter{progress}))
		if err != nil {
			fmt.Println("Error saving episode:", err)
			continue
		}
		close(progress)

		fmt.Printf("\nDownloaded '%s'\n", filename)
	}
}
