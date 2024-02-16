package genius

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/rs/zerolog/log"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	defaultRetryDuration = time.Second * 5
)

// Client is a client for Genius API.
type Client struct {
	AccessToken   string
	baseURL       string
	unofficialUrl string
	client        *http.Client
}

type ClientOption func(client *Client)

// NewClient creates Client to work with Genius API
// You can pass http.Client or it will use http.DefaultClient by default
//
// It requires a token for accessing Genius API.
func NewClient(httpClient *http.Client, token string, opts ...ClientOption) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	c := &Client{AccessToken: token, client: httpClient, baseURL: "https://api.genius.com", unofficialUrl: "https://genius.com/api"}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// WithBaseURL provides an alternative base url to use for requests to the Spotify API. This can be used to connect to a
// staging or other alternative environment.
func WithBaseURL(url string) ClientOption {
	return func(client *Client) {
		client.baseURL = url
	}
}

func retryDuration(resp *http.Response) time.Duration {
	raw := resp.Header.Get("Retry-After")
	if raw == "" {
		return defaultRetryDuration
	}
	seconds, err := strconv.ParseInt(raw, 10, 32)
	if err != nil {
		return defaultRetryDuration
	}
	return time.Duration(seconds) * time.Second
}

// doRequest makes a request and puts authorization token in headers.
func (c *Client) doRequest(req *http.Request) ([]byte, error) {
	req.Header.Set("Authorization", "Bearer "+c.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	for {
		resp, err := c.client.Do(req)
		if err != nil {
			return nil, err
		}

		defer resp.Body.Close()

		if resp.StatusCode == 429 || resp.StatusCode == 1015 {
			time.Sleep(retryDuration(resp))
			continue
			/*
				select {
					//case <-ctx.Done():
						// If the context is cancelled, return the original error
				case <-time.After(retryDuration(resp)):
					fmt.Printf("Retrying url: %s\n", req.URL.Path)
					continue
				}
			*/
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("%s", body)
		}

		return body, nil
	}

	return nil, nil
}

// GetAccount returns current user account data.
func (c *Client) GetAccount() (*GeniusResponse, error) {
	url := fmt.Sprintf(c.baseURL + "/account/")
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	bytes, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}

	var response GeniusResponse
	err = json.Unmarshal(bytes, &response)
	if err != nil {
		return nil, err
	}

	return &response, nil
}

// GetArtist returns Artist object in response
// Uses "dom" as textFormat by default.
func (c *Client) GetArtist(id int) (*GeniusResponse, error) {
	return c.GetArtistDom(id)
}

// GetArtistDom returns Artist object in response
// With "dom" as textFormat.
func (c *Client) GetArtistDom(id int) (*GeniusResponse, error) {
	return c.getArtist(id, "dom")
}

// GetArtistPlain returns Artist object in response
// With "plain" as textFormat.
func (c *Client) GetArtistPlain(id int) (*GeniusResponse, error) {
	return c.getArtist(id, "plain")
}

// GetArtistHTML returns Artist object in response
// With "html" as textFormat.
func (c *Client) GetArtistHTML(id int) (*GeniusResponse, error) {
	return c.getArtist(id, "html")
}

func getPerPage(total int, fetched int, perPage int) int {
	if newPerPage := total - fetched; newPerPage < perPage {
		return newPerPage
	}
	return perPage
}

func (c *Client) GetArtistSongs(id int, sort string, total int) ([]*Song, error) {
	perPage := 50
	var songs []*Song
	page := 1

	// Check if total is 0 and set a flag
	fetchUntilEnd := total == -1

	// Initialize newPerPage only if total is not -1
	var newPerPage int
	if !fetchUntilEnd {
		newPerPage = getPerPage(total, (page-1)*perPage, perPage)
	}

	for fetchUntilEnd || newPerPage > 0 {

		response, err := c.getArtistSongsPage(id, sort, newPerPage, page)
		if err != nil {
			return nil, err
		}

		// Break the loop if NextPage is nil and total is 0
		if fetchUntilEnd && response.Response.NextPage == 0 {
			break
		}

		songs = append(songs, response.Response.Songs...)

		page = response.Response.NextPage
		if !fetchUntilEnd {
			newPerPage = getPerPage(total, (page-1)*perPage, perPage)
		}
	}

	return songs, nil
}

// GetArtistSongs returns array of songs objects in response.
func (c *Client) getArtistSongsPage(id int, sort string, perPage int, page int) (*GeniusResponse, error) {
	url := fmt.Sprintf(c.baseURL+"/artists/%d/songs", id)
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	q.Add("sort", sort)
	q.Add("per_page", strconv.Itoa(perPage))
	q.Add("page", strconv.Itoa(page))
	req.URL.RawQuery = q.Encode()

	bytes, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}

	var response GeniusResponse
	err = json.Unmarshal(bytes, &response)
	if err != nil {
		return nil, err
	}

	return &response, nil
}

func (c *Client) GetSongWithLyrics(id int) (*Song, error) {
	song, err := c.GetSongDom(id)
	if err != nil {
		return nil, err
	}
	lyrics, err := c.GetLyrics(song.URL)
	if err != nil {
		return nil, err
	}
	song.Lyrics = lyrics

	return song, nil
}

// GetSong returns Song object in response
//
// Uses "dom" as textFormat by default.
func (c *Client) GetSong(id int) (*Song, error) {
	return c.GetSongDom(id)
}

// GetSongDom returns Song object in response
// With "dom" as textFormat.
func (c *Client) GetSongDom(id int) (*Song, error) {
	return c.getSong(id, "dom")
}

// GetSongPlain returns Song object in response
// With "plain" as textFormat.
func (c *Client) GetSongPlain(id int) (*Song, error) {
	return c.getSong(id, "plain")
}

// GetSongHTML returns Song object in response
// With "html" as textFormat.
func (c *Client) GetSongHTML(id int) (*Song, error) {
	return c.getSong(id, "html")
}

// GetSong returns Song object in response.
func (c *Client) getSong(id int, textFormat string) (*Song, error) {
	url := fmt.Sprintf(c.baseURL+"/songs/%d", id)
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	q.Add("text_format", textFormat)
	req.URL.RawQuery = q.Encode()

	bytes, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}

	var response GeniusResponse
	err = json.Unmarshal(bytes, &response)
	if err != nil {
		return nil, err
	}

	if response.Response == nil {
		return nil, errors.New("No song found")
	}

	return response.Response.Song, nil
}

func (c *Client) GetArtistAlbums(id int) ([]*Album, error) {
	var albums []*Album
	page := 1
	for page >= 1 {
		response, err := c.getArtistAlbumsPage(id, 50, page)
		if err != nil {
			return nil, err
		}

		page = response.Response.NextPage
		albums = append(albums, response.Response.Albums...)
	}

	return albums, nil
}

func (c *Client) getArtistAlbumsPage(id int, perPage int, page int) (*GeniusResponse, error) {
	getArtistAlbumsURL := fmt.Sprintf(c.unofficialUrl+"/artists/%d/albums", id)
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, getArtistAlbumsURL, nil)
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	q.Add("per_page", strconv.Itoa(perPage))
	q.Add("page", strconv.Itoa(page))
	req.URL.RawQuery = q.Encode()

	bytes, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}

	var response GeniusResponse
	err = json.Unmarshal(bytes, &response)
	if err != nil {
		return nil, err
	}

	return &response, nil
}

// GetAlbum returns Album object in response
func (c *Client) GetAlbum(id int, getTracks bool) (*Album, error) {
	return c.getAlbumDom(id, getTracks)
}

func (c *Client) getAlbumDom(id int, getTracks bool) (*Album, error) {
	return c.getAlbum(id, getTracks, "dom")
}

func (c *Client) getAlbum(id int, getTracks bool, textFormat string) (*Album, error) {
	getAlbumURL := fmt.Sprintf(c.baseURL+"/albums/%d", id)
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, getAlbumURL, nil)
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	q.Add("text_format", textFormat)
	req.URL.RawQuery = q.Encode()

	bytes, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}

	var response GeniusResponse
	err = json.Unmarshal(bytes, &response)
	if err != nil {
		return nil, err
	}

	if getTracks {
		albumTracks, err := c.GetAlbumTracks(id)
		if err != nil {
			return nil, err
		}

		response.Response.Album.Tracks = albumTracks
	}

	return response.Response.Album, nil
}

func (c *Client) GetAlbumTracks(id int) ([]*AlbumTrack, error) {
	var tracks []*AlbumTrack
	page := 1
	for page >= 1 {
		response, err := c.getAlbumTracksPage(id, 50, page)
		if err != nil {
			return nil, err
		}

		page = response.Response.NextPage
		tracks = append(tracks, response.Response.AlbumTracks...)
	}

	return tracks, nil
}

func (c *Client) getAlbumTracksPage(id int, perPage int, page int) (*GeniusResponse, error) {
	getAlbumURL := fmt.Sprintf(c.baseURL+"/albums/%d/tracks", id)
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, getAlbumURL, nil)
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	q.Add("per_page", strconv.Itoa(perPage))
	q.Add("page", strconv.Itoa(page))
	req.URL.RawQuery = q.Encode()

	bytes, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}

	var response GeniusResponse
	err = json.Unmarshal(bytes, &response)
	if err != nil {
		return nil, err
	}

	return &response, nil
}

// getArtist is a method taking id and textFormat as arguments to make request and return Artist object in response.
func (c *Client) getArtist(id int, textFormat string) (*GeniusResponse, error) {
	getArtistURL := fmt.Sprintf(c.baseURL+"/artists/%d", id)
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, getArtistURL, nil)
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	q.Add("text_format", textFormat)
	req.URL.RawQuery = q.Encode()

	bytes, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}

	var response GeniusResponse
	err = json.Unmarshal(bytes, &response)
	if err != nil {
		return nil, err
	}

	return &response, nil
}

// Search returns array of Hit objects in response
//
// Currently only songs are searchable by this handler.
func (c *Client) Search(q string) (*GeniusResponse, error) {
	searchURL := fmt.Sprintf(c.baseURL + "/search")
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, searchURL, nil)
	if err != nil {
		return nil, err
	}

	getParams := req.URL.Query()
	getParams.Add("q", q)
	req.URL.RawQuery = getParams.Encode()

	bytes, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}

	var response GeniusResponse
	err = json.Unmarshal(bytes, &response)
	if err != nil {
		return nil, err
	}

	return &response, nil
}

//https://genius.com/api/page_data/album?page_path=%2Falbums%2FVarious-artists%2FAbove-the-rim-the-soundtrack

func (c *Client) WebSearch(perPage int, searchTerm string) (*GeniusResponse, error) {
	searchURL := fmt.Sprintf(c.baseURL + "/search/multi")

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, searchURL, nil)
	if err != nil {
		return nil, err
	}

	params := url.Values{}
	params.Add("per_page", strconv.Itoa(perPage))
	params.Add("q", searchTerm)
	req.URL.RawQuery = params.Encode()

	bytes, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}

	var response GeniusResponse
	err = json.Unmarshal(bytes, &response)
	if err != nil {
		return nil, err
	}

	return &response, nil
}

// GetAnnotation gets annotation object in response.
func (c *Client) GetAnnotation(id string, textFormat string) (*GeniusResponse, error) {
	annotationsURL := fmt.Sprintf(c.baseURL+"/annotations/%s", id)
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, annotationsURL, nil)
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	q.Add("text_format", textFormat)
	req.URL.RawQuery = q.Encode()

	bytes, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}

	var response GeniusResponse
	err = json.Unmarshal(bytes, &response)
	if err != nil {
		return nil, err
	}

	response.Response.Annotation.Process(textFormat)

	return &response, nil
}

func GetArtistFromSearchResponse(response *GeniusResponse, searchTerm string) (*Song, error) {
	return getItemFromSearchResponse(response, searchTerm, "artist", "name")
}

func GetSongFromSearchResponse(response *GeniusResponse, searchTerm string) (*Song, error) {
	return getItemFromSearchResponse(response, searchTerm, "song", "title")
}

func getItemFromSearchResponse(response *GeniusResponse, searchTerm string, itemType string, resultType string) (*Song, error) {
	var hits []Hit
	for _, section := range response.Response.Sections {
		if section.Type == itemType {
			hits = append(hits, section.Hits...)
		}
	}

	for _, hit := range hits {
		if strings.EqualFold(resultType, "name") {
			if strings.EqualFold(hit.Result.Title, searchTerm) {
				return hit.Result, nil
			}
		}
	}

	if len(hits) < 1 {
		return nil, fmt.Errorf("could not find a match for: %s", searchTerm)
	}
	return hits[0].Result, nil
}

func (c *Client) GetLyrics(uri string) (string, error) {
	var err error
	var req *http.Request
	var res *http.Response

	if req, err = http.NewRequest(http.MethodGet, uri, nil); err != nil {
		return "", err
	}

	if res, err = c.client.Do(req); err != nil {
		return "", err
	}
	//
	defer res.Body.Close()

	bodyBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	lyrics, extractErr := NewExtractor(strings.NewReader(string(bodyBytes))).Extract()
	if extractErr != nil {
		return "", extractErr
	}

	lyrics = strings.TrimSpace(lyrics)

	if strings.HasSuffix(lyrics, "Embed") {
		found := false
		lyrics, found = strings.CutSuffix(lyrics, "Embed")
		if found {
			log.Debug().Msg("Embed found at end of lyrics")
		}
	}

	return lyrics, nil
}
