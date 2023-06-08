package genius_test

import (
	"os"
	"testing"

	"github.com/broxgit/genius"
)

func TestNewClient(t *testing.T) {
	accessToken := os.Getenv("ACCESS_TOKEN")
	client := genius.NewClient(nil, accessToken)

	response, err := client.GetArtistHTML(16775)
	if err != nil {
		t.Fatal("error occurred getting artist", err)
	}

	t.Log(response.Response.Artist)
}
