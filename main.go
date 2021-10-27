package main

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/go-resty/resty/v2"
)

var catMap = map[int]string{
	1:     "Uncategorized",
	9:     "News",
	10:    "Sport",
	14:    "Opinion",
	21:    "Lifestyle",
	26:    "Kaila",
	45:    "Local Travel",
	54:    "Nai Lalakai",
	55:    "Shanti Dut",
	36631: "People",
	56765: "Today’s Main Story",
	64517: "Dining Entertainment",
	89320: "Classifieds",
}

type Article struct {
	Tag     string `json:"tag"`
	Title   string `json:"title"`
	Author  string `json:"author"`
	Url     string `json:"url"`
	Content string `json:"content"`
	Date    string `json:"date"`
}

func main() {

	useCatId := flag.String("cat", "", "Input category id.")
	useSearch := flag.String("search", "", "Input search keywords.")
	useOut := flag.String("out", "", "Output file path.")
	useFormat := flag.String("format", "json", "File format.")
	flag.Parse()

	if len(os.Args) < 2 {
		fmt.Println("category id: ")
		for k, v := range catMap {
			fmt.Println(k, " => ", v)
		}
		fmt.Println("Usage: [command] -cat=1 -search='chinese' -out=data.csv -format=csv")
		os.Exit(0)
	}

	if *useCatId == "" {
		fmt.Println("Must input category id for crawler.")
		os.Exit(0)
	}

	if *useSearch == "" {
		fmt.Println("Must input search keywords for crawler.")
		os.Exit(0)
	}

	if *useOut == "" {
		fmt.Println("Must input file path for crawler.")
		os.Exit(0)
	}

	if *useFormat != "json" && *useFormat != "csv" {
		fmt.Println("The file format only support 'json' and 'csv'.")
		os.Exit(0)
	}

	articles := []Article{}

	client := resty.New()

	var body bytes.Buffer
	body.WriteString("catID=")
	body.WriteString(*useCatId)
	body.WriteString("&search=")
	body.WriteString(*useSearch)

	resp, err := client.R().
		SetHeader("Content-Type", "application/x-www-form-urlencoded").
		SetBody(body.String()).
		Post("https://www.fijitimes.com.fj/wp-content/themes/fijitimes/generate-archive.php")

	if err != nil {
		log.Fatal(err)
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(resp.String()))

	if err != nil {
		log.Fatal(err)
	}

	// Find the review items
	doc.Find(".archive-post-container .archive-post").Each(func(i int, s *goquery.Selection) {
		// For each item found, get the title
		title := s.Find("a").Text()
		url, _ := s.Find("a").Attr("href")
		date := strings.Split(s.Find("p").Text(), "|")[0]
		tag := strings.Split(s.Find("p").Text(), "|")[1]
		article := Article{tag, title, "", url, "", date}
		articles = append(articles, article)
	})

	for i := 0; i < len(articles); i++ {
		article := articles[i]

		fmt.Println(i, "-> ", article.Url)
		resp, err := client.R().Get(article.Url)
		if err != nil {
			log.Fatal(err)
		}
		doc, err := goquery.NewDocumentFromReader(strings.NewReader(resp.String()))
		if err != nil {
			log.Fatal(err)
		}

		content := doc.Find(".single-cat-content").Text()
		author := doc.Find(".header-extras .byline").Text()
		date := doc.Find(".header-extras .section-date").Text()
		articles[i].Author = author
		articles[i].Content = content
		articles[i].Date = date

		fmt.Println(articles[i])
	}

	if len(articles) == 0 {
		fmt.Println("No data.")
		os.Exit(0)
	}

	file, err := os.Create(*useOut)

	if err != nil {
		fmt.Printf("Failed creating file: %s", err)
	}
	defer file.Close()

	// Write UTF-8 BOM, support windows os show chinese
	file.WriteString("\xEF\xBB\xBF")

	if *useFormat == "json" {
		writer := json.NewEncoder(file)
		writer.SetIndent("", "  ")
		writer.Encode(articles)
		return
	}

	if *useFormat == "csv" {
		writer := csv.NewWriter(file)
		writer.Write([]string{"tag", "title", "author", "url", "date", "content"})
		for _, article := range articles {
			writer.Write([]string{article.Tag, article.Title, article.Author, article.Url, article.Date, article.Content})
		}
		writer.Flush()
		return
	}

	fmt.Println("Invalid filename.")
	os.Exit(0)
}
