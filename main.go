package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"

	_ "github.com/mattn/go-sqlite3"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var client = &http.Client{}

var TMBDauth = "Bearer eyJhbGciOiJIUzI1NiJ9.eyJhdWQiOiJlNjZhMmZkNDc5MGIyMmFlNGQxYTYwOTk3ZjhhNDdiMiIsIm5iZiI6MTcyODYwNzMwOC42NjI5MDMsInN1YiI6IjY3MDg3M2MwZWU5NjE0ODU4NzI0OTI0OSIsInNjb3BlcyI6WyJhcGlfcmVhZCJdLCJ2ZXJzaW9uIjoxfQ.FURHYebERepkb23LF7t8V72l7axCq_-C6WKsXRa6lEw"
var qbittorrentUsername = "imad"
var qbittorrentPassword = "AXAV^rJ*AsAdUG7J5mf*sb"

func enableCors(w *http.ResponseWriter) {
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
}

var db *gorm.DB

type Movie struct {
	ID   string `gorm:"primaryKey"`
	Name string
}

func initDB() {
	var err error
	db, err = gorm.Open(sqlite.Open("movies.db"), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}
	db.AutoMigrate(&Movie{})
}

func qbittorrentLogin(username, password string) (string, error) {
	url := "http://localhost:8987/api/v2/auth/login"
	data := fmt.Sprintf("username=%s&password=%s", username, password)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer([]byte(data)))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("login failed, status code: %d", resp.StatusCode)
	}

	for _, cookie := range resp.Cookies() {
		if cookie.Name == "SID" {
			return cookie.Value, nil
		}
	}

	return "", fmt.Errorf("no session ID found")
}

func qbittorrentGetTorrents(sessionID string) error {
	url := "http://localhost:8987/api/v2/torrents/info"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Cookie", fmt.Sprintf("SID=%s", sessionID))
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to get torrents, status code: %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var torrents []map[string]interface{}
	err = json.Unmarshal(body, &torrents)
	if err != nil {
		return err
	}

	for _, torrent := range torrents {
		fmt.Println("Torrent Name:", torrent["name"])
	}

	return nil
}

func TMDBsearchMovie(search string) ([]map[string]interface{}, error) {
	encodedSearch := url.QueryEscape(search)
	apiUrl := fmt.Sprintf("https://api.themoviedb.org/3/search/movie?query=%s", encodedSearch)

	req, _ := http.NewRequest("GET", apiUrl, nil)

	req.Header.Add("accept", "application/json")
	req.Header.Add("Authorization", TMBDauth)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", res.StatusCode, string(body))
	}

	body, _ := io.ReadAll(res.Body)

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("JSON unmarshal error: %v", err)
	}

	results, ok := result["results"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("unable to parse movie results")
	}

	var movies []map[string]interface{}
	for _, movie := range results {
		movieMap := movie.(map[string]interface{})
		if posterPath, exists := movieMap["poster_path"].(string); exists && posterPath != "" {
			movies = append(movies, movieMap)
		}
	}

	return movies, nil
}

func TMDBGetMovieByID(id string) (map[string]interface{}, error) {
	apiUrl := fmt.Sprintf("https://api.themoviedb.org/3/movie/%s", id)

	req, _ := http.NewRequest("GET", apiUrl, nil)
	req.Header.Add("accept", "application/json")
	req.Header.Add("Authorization", TMBDauth)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", res.StatusCode, string(body))
	}

	body, _ := io.ReadAll(res.Body)

	var movieData map[string]interface{}
	if err := json.Unmarshal(body, &movieData); err != nil {
		return nil, fmt.Errorf("JSON unmarshal error: %v", err)
	}

	return movieData, nil
}

func downloadPosterImage(movieID, posterPath string) error {
	posterURL := fmt.Sprintf("https://image.tmdb.org/t/p/original%s", posterPath)

	resp, err := http.Get(posterURL)
	if err != nil {
		return fmt.Errorf("failed to download poster: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to fetch poster, status code: %d", resp.StatusCode)
	}

	imgPath := path.Join("web", "img", fmt.Sprintf("%s.jpg", movieID))
	outFile, err := os.Create(imgPath)
	if err != nil {
		return fmt.Errorf("failed to create image file: %v", err)
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to save image: %v", err)
	}

	return nil
}

func addMovieHandler(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)

	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "Query parameter is missing", http.StatusBadRequest)
		return
	}

	msg := "Movie added successfully"

	movieInfo, err := TMDBGetMovieByID(id)
	if err != nil {
		msg = "Failed to fetch movie info from TMDB"
	}

	movieName, ok := movieInfo["title"].(string)
	if !ok {
		msg = "Failed to parse movie info from TMDB"
	}

	posterPath, posterOk := movieInfo["poster_path"].(string)
	if posterOk {
		if err := downloadPosterImage(id, posterPath); err != nil {
			msg = "Failed to download movie poster"
		}
	} else {
		msg = "Poster path not available from TMDB"
	}

	movie := Movie{ID: id, Name: movieName}
	if err := db.Create(&movie).Error; err != nil {
		msg = "Failed to add movie to database"
	}

	response := map[string]string{
		"message": msg,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func getMoviesHandler(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)

	var movies []Movie

	if err := db.Find(&movies).Error; err != nil {
		http.Error(w, "Failed to retrieve movies from database", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(movies)
}

func TMDBsearchMovieHandler(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)

	query := r.URL.Query().Get("query")
	if query == "" {
		http.Error(w, "Query parameter is missing", http.StatusBadRequest)
		return
	}

	movies, err := TMDBsearchMovie(query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(movies)
}

func main() {
	// sessionId, err := qbittorrentLogin(qbittorrentUsername, qbittorrentPassword)
	// if err != nil {
	//     log.Fatal("Login error:", err)
	// }
	// err = qbittorrentGetTorrents(sessionId)

	///////////////

	initDB()

	fs := http.FileServer(http.Dir("./web"))
	http.Handle("/", fs)

	http.HandleFunc("/api/search", TMDBsearchMovieHandler)
	http.HandleFunc("/api/add_movie", addMovieHandler)
	http.HandleFunc("/api/get_movies", getMoviesHandler)

	log.Println("Listening on :8080...")
	err := http.ListenAndServe("localhost:8080", nil)
	if err != nil {
		log.Fatal(err)
	}
}
