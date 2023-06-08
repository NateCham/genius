package genius

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/rs/zerolog/log"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	common "github.com/broxgit/common/http"
)

const (
	baseURL string = "https://api.genius.com"
)

// Client is a client for Genius API.
type Client struct {
	AccessToken string
	client      *common.HTTPRetry
}

// NewClient creates Client to work with Genius API
// You can pass http.Client or it will use http.DefaultClient by default
//
// It requires a token for accessing Genius API.
func NewClient(httpClient *common.HTTPRetry, token string) *Client {
	if httpClient == nil {
		httpClient = common.NewHTTPRetry()
	}

	c := &Client{AccessToken: token, client: httpClient}
	return c
}

// doRequest makes a request and puts authorization token in headers.
func (c *Client) doRequest(req *http.Request) ([]byte, error) {
	req.Header.Set("Authorization", "Bearer "+c.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s", body)
	}

	return body, nil
}

// GetAccount returns current user account data.
func (c *Client) GetAccount() (*GeniusResponse, error) {
	url := fmt.Sprintf(baseURL + "/account/")
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

// GetArtistSongs returns array of songs objects in response.
func (c *Client) GetArtistSongs(id int, sort string, perPage int, page int) (*GeniusResponse, error) {
	url := fmt.Sprintf(baseURL+"/artists/%d/songs", id)
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
	url := fmt.Sprintf(baseURL+"/songs/%d", id)
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

	return response.Response.Song, nil
}

// getArtist is a method taking id and textFormat as arguments to make request and return Artist object in response.
func (c *Client) getArtist(id int, textFormat string) (*GeniusResponse, error) {
	getArtistURL := fmt.Sprintf(baseURL+"/artists/%d", id)
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
	searchURL := fmt.Sprintf(baseURL + "/search")
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

func WebSearch(perPage int, searchTerm string) (GeniusResponse, error) {
	webSearchURL := "https://genius.com/api/search/multi?"

	params := url.Values{}
	params.Add("per_page", strconv.Itoa(perPage))
	params.Add("q", searchTerm)

	requestURL, _ := url.ParseRequestURI(webSearchURL)
	requestURL.RawQuery = params.Encode()
	searchURL := fmt.Sprintf("%v", requestURL)

	var target GeniusResponse

	response, err := http.Get(searchURL)
	if err != nil {
		return target, err
	}

	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return target, err
	}

	if err = json.Unmarshal(body, &target); err != nil {
		return target, err
	}

	return target, nil
}

// GetAnnotation gets annotation object in response.
func (c *Client) GetAnnotation(id string, textFormat string) (*GeniusResponse, error) {
	annotationsURL := fmt.Sprintf(baseURL+"/annotations/%s", id)
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

	if res, err = c.client.HTTPClient.Do(req); err != nil {
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
