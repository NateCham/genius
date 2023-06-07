package genius_test

import (
	"fmt"

	"github.com/broxgit/genius-api/genius/genius"
)

func ExampleNewClient() {
	accessToken := "token"
	client := genius.NewClient(nil, accessToken)

	response, err := client.GetArtistHTML(16775)
	if err != nil {
		panic(err)
	}

	fmt.Println(response.Response.Artist)
}
