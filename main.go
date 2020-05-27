package main

import (
	"html/template"
	"net/http"
	"os"
	"fmt"
	"net/url"
	"flag"
	"time"
	"log"
	"strconv"
	"encoding/json"
	"math"

)

// tpl is a package level var , points to a template definition
// wrap the invocation of template.ParseFiles with template.Must so that the code panics if an error is obtained. 
var tpl = template.Must(template.ParseFiles("index.html"))
var apiKey *string

// Data model - convert json to struct from JSON-to-GO
type Source struct {
	ID   interface{} `json:"id"`
	Name string      `json:"name"`
} 

type Articles struct {
	Source 	Source	`json:"source"`
	Author      string    `json:"author"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	URL         string    `json:"url"`
	URLToImage  string    `json:"urlToImage"`
	PublishedAt time.Time `json:"publishedAt"`
	Content     string    `json:"content"`
} 

func (a *Articles) FormatPublishedDate() string {
	year, month, day := a.PublishedAt.Date()
	return fmt.Sprintf("%v %d, %d", month, day, year)
}

type Results struct {
	Status       string `json:"status"`
	TotalResults int    `json:"totalResults"`
	Articles []Articles `json:"articles"`
}

type Search struct {
	SearchKey  string
	NextPage   int
	TotalPages int
	Results    Results
}

// check if next page field is greater than total page 
func (s *Search) IsLastPage() bool {
	return s.NextPage >= s.TotalPages
}

// keep track of current page
func (s *Search) CurrentPage() int {
	if s.NextPage == 1 {
		return s.NextPage
	}

	return s.NextPage - 1
}
// method for previous button
func (s *Search) PreviousPage() int {
	return s.CurrentPage() - 1
}

// execute the template created 
func indexHandler(w http.ResponseWriter, r *http.Request) {
	tpl.Execute(w, nil)
}

type NewsAPIError struct {
	Status  string `json:"status"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

func searchHandler(w http.ResponseWriter, r *http.Request) {

	u, err := url.Parse(r.URL.String())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal server error"))
		return
	}

	params := u.Query()
	searchKey := params.Get("q")
	page := params.Get("page")
	if page == "" {
		page = "1"
	}

	search := &Search{}
	search.SearchKey = searchKey

	next, err := strconv.Atoi(page)
	if err != nil {
		http.Error(w, "Unexpected server error", http.StatusInternalServerError)
		return
	}

	search.NextPage = next
	pageSize := 20

	endpoint := fmt.Sprintf("https://newsapi.org/v2/everything?q=%s&pageSize=%d&page=%d&apiKey=%s&sortBy=publishedAt&language=en", url.QueryEscape(search.SearchKey), pageSize, search.NextPage, *apiKey)
	resp, err := http.Get(endpoint)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	defer resp.Body.Close()

	// error handling
	if resp.StatusCode != 200 {
		newError := &NewsAPIError{}
		err := json.NewDecoder(resp.Body).Decode(newError)
		if err != nil {
		  http.Error(w, "Unexpected server error", http.StatusInternalServerError)
		  return
		}
	  
		http.Error(w, newError.Message, http.StatusInternalServerError)
		return
	  }

	err = json.NewDecoder(resp.Body).Decode(&search.Results)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	search.TotalPages = int(math.Ceil(float64(search.Results.TotalResults / pageSize)))
	// if next page is rendered , increment next page
	if ok := !search.IsLastPage(); ok {
		search.NextPage++
	}

	err = tpl.Execute(w, search)
	if err != nil {
		log.Println(err)
	}
}

func main() {
	//define a string flag  - (flagname, default value, usage description)
	apiKey = flag.String("apikey", "", "Newsapi.org access key")
	// parse the key
	flag.Parse()

	if *apiKey == "" {
		log.Fatal("apiKey must be set")
	}


	port := os.Getenv("PORT")
	if port == "" {
		port = "2000"
	}

	/* creates new HTTP request multiplexer and assigns it to mux -
	a request multiplexer matches the URL of incoming requests against a list 
	of registered paths and calls the associated handler for the path whenever a match is found */
	mux := http.NewServeMux()


	// create one handler to take care of serving all static assets.
	fs := http.FileServer(http.Dir("assets"))

	//direct the router to use this file server object for all paths beginning with the /assets/ prefix
	mux.Handle("/assets/", http.StripPrefix("/assets/", fs))

	// direct urls with /search
	mux.HandleFunc("/search", searchHandler)

	// register handler function for the root path '/' and 
	//second argument - handler fuction taking in the request and writing the response
	mux.HandleFunc("/", indexHandler)

	//starts the server on defined port
	http.ListenAndServe(":"+port, mux)
}
