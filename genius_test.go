package genius_test

import (
	"os"
	"strings"
	"testing"

	"github.com/natecham/genius"
)

func TestNewClient(t *testing.T) {
	accessToken := os.Getenv("ACCESS_TOKEN")
	client := genius.NewClient(nil, accessToken)

	response, err := client.GetArtistHTML(1177)
	if err != nil {
		t.Fatal("error occurred getting artist", err)
	}

	if !strings.EqualFold(response.Response.Artist.Name, "Taylor Swift") {
		t.Fatal("Unexpected artist, wanted Taylor Swift, got ", response.Response.Artist.Name)
	}

	t.Log(response.Response.Artist.Name)

	song, err := client.GetSong(57418)
	if err != nil {
		t.Fatal("error occurred getting song", err)
	}

	lyrics, lyricsErr := client.GetLyrics(song.URL)
	if lyricsErr != nil {
		t.Fatal("error occurred getting lyrics", lyricsErr)
	}

	if !strings.Contains(lyrics, "You're not sorry") {
		t.Fatal("lyrics missing")
	}

	song2, lyErr := client.GetSongWithLyrics(57418)
	if lyErr != nil {
		t.Fatal("error getting lyrics with GetSongWithLyrics")
	}

	if !strings.Contains(song2.Lyrics, "You're not sorry") {
		t.Fatal("lyrics missing")
	}

	if !strings.EqualFold(song2.Lyrics, lyrics) {
		t.Fatal("lyrics don't match :(")
	}

}
